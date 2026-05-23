package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type ImportHandler struct {
	DB *sql.DB
}

var categoryMap = map[string]string{
	"梦幻精品粽子": "消耗品", "金柳露": "消耗品", "C66": "消耗品",
	"净瓶玉露": "消耗品", "超级净瓶玉露": "消耗品", "分解符": "消耗品",
	"彩果": "消耗品", "强化石": "消耗品", "碧藕": "消耗品",
	"储灵袋": "消耗品", "月花露": "消耗品", "摇钱树树苗": "消耗品",
	"彩虹草": "消耗品", "红玫瑰": "消耗品", "海马": "消耗品",
	"修炼果": "其他",
}

func getCategory(name string) string {
	if c, ok := categoryMap[name]; ok {
		return c
	}
	prefixRules := []struct{ prefix, cat string }{
		{"炼妖石", "炼妖石"}, {"图册", "图册"}, {"如意丹", "如意丹"},
		{"珍珠", "珍珠"}, {"附魔宝珠", "附魔宝珠"}, {"种子", "消耗品"},
	}
	for _, r := range prefixRules {
		if strings.HasPrefix(name, r.prefix) {
			return r.cat
		}
	}
	switch name {
	case "金刚石", "定魂珠", "夜光珠", "龙鳞", "避水珠":
		return "宝石"
	case "太阳石", "舍利子", "月亮石", "黑宝石", "红玛瑙", "光芒石":
		return "装备石"
	case "战魄":
		return "书铁"
	case "仙露小丸子":
		return "其他"
	}
	if strings.HasPrefix(name, "书") || strings.HasPrefix(name, "铁") {
		return "书铁"
	}
	if strings.HasSuffix(name, "元宵") {
		return "元宵"
	}
	if strings.HasSuffix(name, "阵") {
		return "阵法"
	}
	if strings.Contains(name, "内丹") || name == "精岳" || name == "矫健" ||
		name == "迅敏" || name == "双星爆" || name == "腾挪劲" {
		return "其他"
	}
	return "兽诀"
}

func excelSerialToDate(serial float64) string {
	baseTime, _ := time.Parse("2006-01-02", "1899-12-30")
	return baseTime.AddDate(0, 0, int(serial)).Format("2006-01-02")
}

var priceRegex = regexp.MustCompile(`[\d.]+`)

func parseCellPrice(cellVal string) (float64, bool) {
	cellVal = strings.TrimSpace(cellVal)
	if cellVal == "" {
		return 0, false
	}
	if num, err := strconv.ParseFloat(cellVal, 64); err == nil && num > 0 {
		return num, true
	}
	numStr := priceRegex.FindString(cellVal)
	if numStr == "" {
		return 0, false
	}
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, false
	}
	return num, true
}

func (h *ImportHandler) ImportExcel(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, "请上传文件", http.StatusBadRequest)
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		writeError(w, "无法解析Excel文件: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil || len(rows) == 0 {
		writeError(w, "Excel文件为空或无法读取", http.StatusBadRequest)
		return
	}

	// Load category ID map
	catIDMap := make(map[string]int)
	catRows, _ := h.DB.Query("SELECT id, name FROM categories")
	if catRows != nil {
		defer catRows.Close()
		for catRows.Next() {
			var id int
			var name string
			catRows.Scan(&id, &name)
			catIDMap[name] = id
		}
	}

	// Parse date columns from header
	type dateCol struct {
		ColIndex int
		Date     string
	}
	var dateCols []dateCol
	headerRow := rows[0]
	for i, cell := range headerRow {
		if i == 0 {
			continue
		}
		// Try MM-DD-YY format (e.g. "01-01-21")
		t, err := time.Parse("01-02-06", cell)
		if err == nil && t.Year() >= 2020 && t.Year() <= 2035 {
			dateCols = append(dateCols, dateCol{ColIndex: i, Date: t.Format("2006-01-02")})
			continue
		}
		// Try Excel serial number
		serial, err := strconv.ParseFloat(cell, 64)
		if err != nil {
			continue
		}
		dateStr := excelSerialToDate(serial)
		t, err = time.Parse("2006-01-02", dateStr)
		if err == nil && t.Year() >= 2020 && t.Year() <= 2035 {
			dateCols = append(dateCols, dateCol{ColIndex: i, Date: dateStr})
		}
	}
	log.Printf("Excel import: found %d date columns", len(dateCols))

	if len(dateCols) == 0 {
		writeError(w, "未找到有效的日期列", http.StatusBadRequest)
		return
	}

	// Import in transaction
	tx, err := h.DB.Begin()
	if err != nil {
		writeError(w, "数据库错误", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	stmtProd, _ := tx.Prepare("INSERT OR IGNORE INTO products (name, category_id, remark) VALUES (?, ?, '')")
	defer stmtProd.Close()
	stmtPrice, _ := tx.Prepare("INSERT OR REPLACE INTO prices (product_id, price_date, price) VALUES (?, ?, ?)")
	defer stmtPrice.Close()

	productCount := 0
	priceCount := 0

	for _, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}
		name := strings.TrimSpace(row[0])
		if name == "" {
			continue
		}

		catName := getCategory(name)
		catID := catIDMap[catName]

		// Get or create product
		var productID int
		err := tx.QueryRow("SELECT id FROM products WHERE name = ?", name).Scan(&productID)
		if err == sql.ErrNoRows {
			res, err := stmtProd.Exec(name, catID)
			if err != nil {
				log.Printf("Warning: insert product %s: %v", name, err)
				continue
			}
			id, _ := res.LastInsertId()
			productID = int(id)
			productCount++
		}

		for _, dc := range dateCols {
			if dc.ColIndex >= len(row) {
				continue
			}
			priceVal, ok := parseCellPrice(row[dc.ColIndex])
			if !ok {
				continue
			}
			stmtPrice.Exec(productID, dc.Date, priceVal)
			priceCount++
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, "导入失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Excel import done: %d products, %d prices", productCount, priceCount)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"imported_products":%d,"imported_prices":%d}`, productCount, priceCount)
}
