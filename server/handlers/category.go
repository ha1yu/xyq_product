package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type CategoryHandler struct {
	DB *sql.DB
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id, name, sort_order FROM categories ORDER BY sort_order")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Category struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
	}

	var cats []Category
	for rows.Next() {
		var c Category
		rows.Scan(&c.ID, &c.Name, &c.SortOrder)
		cats = append(cats, c)
	}
	if cats == nil {
		cats = []Category{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cats)
}
