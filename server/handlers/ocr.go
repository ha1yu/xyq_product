package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type OcrHandler struct {
	Endpoint string
	Model    string
}

type ocrRequest struct {
	Image string `json:"image"`
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

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeError(w, "AI模型请求失败: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, "读取响应失败", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != 200 {
		writeError(w, "AI模型返回错误: "+string(respBody), resp.StatusCode)
		return
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &chatResp); err != nil || len(chatResp.Choices) == 0 {
		writeError(w, "解析AI响应失败", http.StatusInternalServerError)
		return
	}

	content := chatResp.Choices[0].Message.Content
	log.Printf("OCR raw content: %q", content)

	items := parseOcrResult(content)
	log.Printf("OCR parsed items: %d items", len(items))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"raw":   content,
		"items": items,
	})
}

func parseOcrResult(content string) []OcrItem {
	// Strategy 1: If we have both [ and ], try fixing missing commas and parsing the whole array.
	if start := strings.Index(content, "["); start >= 0 {
		if end := strings.LastIndex(content, "]"); end > start {
			fixed := fixMissingCommas(content[start : end+1])
			var arr []struct {
				Name  string  `json:"name"`
				Price float64 `json:"price"`
			}
			if err := json.Unmarshal([]byte(fixed), &arr); err == nil && len(arr) > 0 {
				items := make([]OcrItem, 0, len(arr))
				for _, it := range arr {
					if it.Name != "" {
						items = append(items, OcrItem{Name: it.Name, Price: it.Price})
					}
				}
				if len(items) > 0 {
					return items
				}
			}
		}
	}

	// Strategy 2: Extract each { ... } object individually and parse separately.
	// AI models often omit commas between objects AND may omit the closing ].
	// Individual objects are always valid JSON on their own.
	var items []OcrItem
	reObj := regexp.MustCompile(`\{[^}]*\}`)
	for _, m := range reObj.FindAllString(content, -1) {
		var obj struct {
			Name  string  `json:"name"`
			Price float64 `json:"price"`
		}
		if err := json.Unmarshal([]byte(m), &obj); err != nil {
			continue
		}
		if obj.Name != "" {
			items = append(items, OcrItem{Name: obj.Name, Price: obj.Price})
		}
	}

	if items == nil {
		return []OcrItem{}
	}
	return items
}

func fixMissingCommas(s string) string {
	return regexp.MustCompile(`}\s+{`).ReplaceAllString(s, "},{")
}

