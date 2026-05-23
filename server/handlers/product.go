package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/mhxy/price-tracker/middleware"
)

type ProductHandler struct {
	DB *sql.DB
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	query := `SELECT p.id, p.name, p.category_id, c.name as category_name, p.remark, p.created_at, p.updated_at,
		(SELECT price FROM prices WHERE product_id = p.id ORDER BY price_date DESC LIMIT 1) as latest_price,
		(SELECT price_date FROM prices WHERE product_id = p.id ORDER BY price_date DESC LIMIT 1) as latest_date
		FROM products p LEFT JOIN categories c ON p.category_id = c.id WHERE 1=1`
	args := []interface{}{}

	if catID := r.URL.Query().Get("category_id"); catID != "" {
		query += " AND p.category_id = ?"
		args = append(args, catID)
	}
	if keyword := r.URL.Query().Get("keyword"); keyword != "" {
		query += " AND p.name LIKE ?"
		args = append(args, "%"+keyword+"%")
	}
	query += " ORDER BY p.id"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Product struct {
		ID           int     `json:"id"`
		Name         string  `json:"name"`
		CategoryID   int     `json:"category_id"`
		CategoryName string  `json:"category_name"`
		Remark       string  `json:"remark"`
		LatestPrice  *float64 `json:"latestPrice"`
		LatestDate   *string  `json:"latestDate"`
		CreatedAt    string  `json:"created_at"`
		UpdatedAt    string  `json:"updated_at"`
	}

	var products []Product
	for rows.Next() {
		var p Product
		rows.Scan(&p.ID, &p.Name, &p.CategoryID, &p.CategoryName, &p.Remark, &p.CreatedAt, &p.UpdatedAt, &p.LatestPrice, &p.LatestDate)
		// Format latestDate to just the date part
		if p.LatestDate != nil && len(*p.LatestDate) >= 10 {
			d := (*p.LatestDate)[:10]
			p.LatestDate = &d
		}
		products = append(products, p)
	}
	if products == nil {
		products = []Product{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		CategoryID int    `json:"category_id"`
		Remark     string `json:"remark"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
		return
	}

	res, err := h.DB.Exec("INSERT INTO products (name, category_id, remark) VALUES (?, ?, ?)",
		req.Name, req.CategoryID, req.Remark)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r.URL.Path, "/api/products/")
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Name       string `json:"name"`
		CategoryID int    `json:"category_id"`
		Remark     string `json:"remark"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec("UPDATE products SET name=?, category_id=?, remark=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
		req.Name, req.CategoryID, req.Remark, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r.URL.Path, "/api/products/")
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec("DELETE FROM products WHERE id=?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *ProductHandler) GetPrices(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r.URL.Path, "/api/products/")
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	rows, err := h.DB.Query("SELECT id, product_id, price_date, price FROM prices WHERE product_id=? ORDER BY price_date", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Price struct {
		ID        int     `json:"id"`
		ProductID int     `json:"product_id"`
		PriceDate string  `json:"price_date"`
		Price     float64 `json:"price"`
	}

	var prices []Price
	for rows.Next() {
		var p Price
		var rawDate string
		rows.Scan(&p.ID, &p.ProductID, &rawDate, &p.Price)
		if len(rawDate) >= 10 {
			p.PriceDate = rawDate[:10]
		} else {
			p.PriceDate = rawDate
		}
		prices = append(prices, p)
	}
	if prices == nil {
		prices = []Price{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prices)
}

func (h *ProductHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/products", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.List(w, r)
		case http.MethodPost:
			middleware.AuthMiddleware(h.Create)(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/products/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/prices") {
			if r.Method == http.MethodGet {
				h.GetPrices(w, r)
				return
			}
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.GetPrices(w, r)
		case http.MethodPut:
			middleware.AuthMiddleware(h.Update)(w, r)
		case http.MethodDelete:
			middleware.AuthMiddleware(h.Delete)(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func extractID(path, prefix string) (int, error) {
	sub := strings.TrimPrefix(path, prefix)
	parts := strings.SplitN(sub, "/", 2)
	return strconv.Atoi(parts[0])
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
