package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"github.com/mhxy/price-tracker/middleware"
)

type PriceHandler struct {
	DB *sql.DB
}

func (h *PriceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProductID int     `json:"product_id"`
		PriceDate string  `json:"price_date"`
		Price     float64 `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request", http.StatusBadRequest)
		return
	}

	res, err := h.DB.Exec("INSERT OR REPLACE INTO prices (product_id, price_date, price) VALUES (?, ?, ?)",
		req.ProductID, req.PriceDate, req.Price)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

func (h *PriceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/prices/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "invalid id", http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec("DELETE FROM prices WHERE id=?", id)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *PriceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/prices/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			middleware.AuthMiddleware(h.Delete)(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/prices", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			middleware.AuthMiddleware(h.Create)(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
}

func (h *PriceHandler) Trend(w http.ResponseWriter, r *http.Request) {
	productIDs := r.URL.Query().Get("product_ids")
	if productIDs == "" {
		writeError(w, "product_ids required", http.StatusBadRequest)
		return
	}

	type TrendDataPoint struct {
		Date  string  `json:"date"`
		Price float64 `json:"price"`
	}

	type TrendSeries struct {
		ProductID   int              `json:"product_id"`
		ProductName string           `json:"product_name"`
		Data        []TrendDataPoint `json:"data"`
	}

	var series []TrendSeries

	ids := splitAndParseIDs(productIDs)
	for _, id := range ids {
		var name string
		err := h.DB.QueryRow("SELECT name FROM products WHERE id=?", id).Scan(&name)
		if err != nil {
			continue
		}

		rows, err := h.DB.Query("SELECT price_date, price FROM prices WHERE product_id=? ORDER BY price_date", id)
		if err != nil {
			continue
		}

		var data []TrendDataPoint
		for rows.Next() {
			var d TrendDataPoint
			var rawDate string
			rows.Scan(&rawDate, &d.Price)
			// Format date: extract just the date part
			d.Date = formatDate(rawDate)
			data = append(data, d)
		}
		rows.Close()
		if data == nil {
			data = []TrendDataPoint{}
		}

		series = append(series, TrendSeries{
			ProductID:   id,
			ProductName: name,
			Data:        data,
		})
	}

	if series == nil {
		series = []TrendSeries{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(series)
}

func formatDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func splitAndParseIDs(s string) []int {
	var ids []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if id, err := strconv.Atoi(part); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func writeError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"error":"%s"}`, msg)
}

func (h *PriceHandler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Items []struct {
			ProductID int     `json:"product_id"`
			PriceDate string  `json:"price_date"`
			Price     float64 `json:"price"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if len(req.Items) == 0 {
		writeError(w, "items required", http.StatusBadRequest)
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR REPLACE INTO prices (product_id, price_date, price) VALUES (?, ?, ?)")
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	inserted := 0
	for _, item := range req.Items {
		if item.ProductID == 0 || item.Price == 0 || item.PriceDate == "" {
			continue
		}
		_, err := stmt.Exec(item.ProductID, item.PriceDate, item.Price)
		if err == nil {
			inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"inserted": inserted,
	})
}
