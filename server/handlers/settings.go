package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type SettingsHandler struct {
	DB *sql.DB
}

func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT key, value FROM settings")
	if err != nil {
		writeError(w, "查询设置失败", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			continue
		}
		result[k] = v
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request", http.StatusBadRequest)
		return
	}

	allowed := map[string]bool{"ocr_endpoint": true, "ocr_model": true}
	for k, v := range req {
		if !allowed[k] || v == "" {
			continue
		}
		if _, err := h.DB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", k, v); err != nil {
			writeError(w, "保存设置失败", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

func (h *SettingsHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		writeError(w, "请填写完整", http.StatusBadRequest)
		return
	}

	username, _ := r.Context().Value("username").(string)
	if username == "" {
		writeError(w, "未登录", http.StatusUnauthorized)
		return
	}

	var hashed string
	if err := h.DB.QueryRow("SELECT password FROM admins WHERE username = ?", username).Scan(&hashed); err != nil {
		writeError(w, "用户不存在", http.StatusNotFound)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(req.OldPassword)); err != nil {
		writeError(w, "旧密码错误", http.StatusBadRequest)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, "密码加密失败", http.StatusInternalServerError)
		return
	}

	if _, err := h.DB.Exec("UPDATE admins SET password = ? WHERE username = ?", string(newHash), username); err != nil {
		writeError(w, "更新密码失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}
