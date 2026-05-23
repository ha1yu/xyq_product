package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type OcrHandler struct {
	Endpoint string // e.g. http://192.168.0.105:7890/v1/chat/completions
	Model    string // e.g. glm-ocr
}

type ocrRequest struct {
	Image string `json:"image"` // base64 encoded image
}

type OcrItem struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func (h *OcrHandler) Recognize(w http.ResponseWriter, r *http.Request) {
	var req ocrRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Image == "" {
		writeError(w, "image required", http.StatusBadRequest)
		return
	}

	// Call local AI model
	prompt := `请识别这张图片中所有梦幻西游游戏相关的商品名称和对应价格（单位：万两）。请严格按照以下JSON数组格式输出，不要输出其他内容：
[{"name":"商品名","price":价格数字}]
如果没有识别到商品和价格，请输出空数组 []`

	payload := map[string]interface{}{
		"model": h.Model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{"type": "text", "text": prompt},
					{"type": "image_url", "image_url": map[string]string{"url": req.Image}},
				},
			},
		},
		"max_tokens": 1024,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		writeError(w, "marshal error", http.StatusInternalServerError)
		return
	}

	httpReq, err := http.NewRequest("POST", h.Endpoint, bytes.NewReader(body))
	if err != nil {
		writeError(w, "create request error", http.StatusInternalServerError)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeError(w, "AI model request failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, "read response error", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != 200 {
		writeError(w, "AI model error: "+string(respBody), resp.StatusCode)
		return
	}

	// Parse OpenAI-compatible response
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &chatResp); err != nil || len(chatResp.Choices) == 0 {
		writeError(w, "parse AI response error", http.StatusInternalServerError)
		return
	}

	content := chatResp.Choices[0].Message.Content
	items := parseOcrResult(content)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"raw":   content,
		"items": items,
	})
}

func parseOcrResult(content string) []OcrItem {
	// Try to extract JSON array from the response
	// Model may wrap it in ```json ... ``` or add extra text
	var items []OcrItem

	// Find JSON array in content
	jsonStr := extractJSONArray(content)
	if jsonStr == "" {
		return items
	}

	// Try parsing as array of objects
	var rawItems []struct {
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &rawItems); err != nil {
		// Try alternate field names
		var altItems []struct {
			Name      string  `json:"商品名"`
			Price     float64 `json:"价格"`
			PriceStr  string  `json:"price"`
			NameAlt   string  `json:"商品名称"`
			PriceAlt  string  `json:"价格（万两）"`
		}
		if err2 := json.Unmarshal([]byte(jsonStr), &altItems); err2 != nil {
			return items
		}
		for _, it := range altItems {
			name := it.Name
			if name == "" {
				name = it.NameAlt
			}
			price := it.Price
			if price == 0 && it.PriceStr != "" {
				fmt.Sscanf(it.PriceStr, "%f", &price)
			}
			if price == 0 && it.PriceAlt != "" {
				fmt.Sscanf(it.PriceAlt, "%f", &price)
			}
			if name != "" {
				items = append(items, OcrItem{Name: name, Price: price})
			}
		}
		return items
	}

	for _, it := range rawItems {
		if it.Name != "" {
			items = append(items, OcrItem{Name: it.Name, Price: it.Price})
		}
	}
	return items
}

func extractJSONArray(s string) string {
	// First try to find content between ```json and ```
	re := regexp.MustCompile("(?s)```(?:json)?\\s*\\n?(\\[.*?\\])\\s*\\n?```")
	if matches := re.FindStringSubmatch(s); len(matches) > 1 {
		return matches[1]
	}

	// Find first [ ... ] block
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return ""
}
