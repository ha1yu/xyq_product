package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/mhxy/price-tracker/middleware"
	"github.com/xuri/excelize/v2"
)

type BackupHandler struct {
	DB *sql.DB
}

func (h *BackupHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/backup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			middleware.AuthMiddleware(h.Export)(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/backup/restore", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			middleware.AuthMiddleware(h.Restore)(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
}

func (h *BackupHandler) Export(w http.ResponseWriter, r *http.Request) {
	f := excelize.NewFile()
	defer f.Close()

	sheet1 := "categories"
	f.SetSheetName("Sheet1", sheet1)

	// Sheet 1: categories
	f.SetCellValue(sheet1, "A1", "category_id")
	f.SetCellValue(sheet1, "B1", "category_name")
	f.SetCellValue(sheet1, "C1", "sort_order")

	catRows, err := h.DB.Query("SELECT id, name, sort_order FROM categories ORDER BY sort_order, id")
	if err != nil {
		writeError(w, "查询分类失败", http.StatusInternalServerError)
		return
	}
	defer catRows.Close()

	row := 2
	type catInfo struct {
		ID   int
		Name string
	}
	var cats []catInfo
	for catRows.Next() {
		var id, sortOrder int
		var name string
		catRows.Scan(&id, &name, &sortOrder)
		f.SetCellValue(sheet1, fmt.Sprintf("A%d", row), id)
		f.SetCellValue(sheet1, fmt.Sprintf("B%d", row), name)
		f.SetCellValue(sheet1, fmt.Sprintf("C%d", row), sortOrder)
		cats = append(cats, catInfo{ID: id, Name: name})
		row++
	}

	// Sheet 2: data (products + prices)
	sheet2 := "data"
	idx, _ := f.NewSheet(sheet2)

	// Collect all unique dates
	dateRows, err := h.DB.Query("SELECT DISTINCT price_date FROM prices ORDER BY price_date")
	if err != nil {
		writeError(w, "查询日期失败", http.StatusInternalServerError)
		return
	}
	var dates []string
	for dateRows.Next() {
		var d string
		dateRows.Scan(&d)
		dates = append(dates, d)
	}
	dateRows.Close()

	// Header row
	headers := []string{"分类", "商品名称", "备注"}
	for _, d := range dates {
		headers = append(headers, d)
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if i >= 3 {
			f.SetCellStr(sheet2, cell, h)
		} else {
			f.SetCellValue(sheet2, cell, h)
		}
	}

	// Query products with category name
	prodRows, err := h.DB.Query(`
		SELECT p.id, p.name, c.name, p.remark
		FROM products p
		JOIN categories c ON p.category_id = c.id
		ORDER BY c.sort_order, p.id
	`)
	if err != nil {
		writeError(w, "查询商品失败", http.StatusInternalServerError)
		return
	}
	defer prodRows.Close()

	type prodInfo struct {
		ID       int
		Name     string
		CatName  string
		Remark   string
	}
	var prods []prodInfo
	for prodRows.Next() {
		var p prodInfo
		prodRows.Scan(&p.ID, &p.Name, &p.CatName, &p.Remark)
		prods = append(prods, p)
	}

	// Build price map: product_id -> date -> price
	priceMap := make(map[int]map[string]float64)
	if len(dates) > 0 {
		priceRows, err := h.DB.Query("SELECT product_id, price_date, price FROM prices")
		if err == nil {
			for priceRows.Next() {
				var pid int
				var date string
				var price float64
				priceRows.Scan(&pid, &date, &price)
				if priceMap[pid] == nil {
					priceMap[pid] = make(map[string]float64)
				}
				priceMap[pid][date] = price
			}
			priceRows.Close()
		}
	}

	// Write product rows
	writtenPrices := 0
	for i, p := range prods {
		r := i + 2
		cell, _ := excelize.CoordinatesToCellName(1, r)
		f.SetCellValue(sheet2, cell, p.CatName)
		cell, _ = excelize.CoordinatesToCellName(2, r)
		f.SetCellValue(sheet2, cell, p.Name)
		cell, _ = excelize.CoordinatesToCellName(3, r)
		f.SetCellValue(sheet2, cell, p.Remark)

		pm := priceMap[p.ID]
		for j, d := range dates {
			if price, ok := pm[d]; ok {
				cell, _ = excelize.CoordinatesToCellName(j+4, r)
				f.SetCellValue(sheet2, cell, price)
				writtenPrices++
			}
		}
	}

	// Set active sheet to data
	f.SetActiveSheet(idx)

	// Generate filename with timestamp
	filename := fmt.Sprintf("backup_%s.xlsx", time.Now().Format("20060102_150405"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	if err := f.Write(w); err != nil {
		log.Printf("Backup export write error: %v", err)
	}
	log.Printf("Backup exported: %d categories, %d products, %d dates, %d price cells", len(cats), len(prods), len(dates), writtenPrices)
}

func (h *BackupHandler) Restore(w http.ResponseWriter, r *http.Request) {
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

	// Validate sheet names
	sheet1 := f.GetSheetName(0)
	if sheet1 != "categories" {
		writeError(w, "无效的备份文件：缺少 categories 工作表", http.StatusBadRequest)
		return
	}
	sheet2Idx := -1
	for i := 1; i < f.SheetCount; i++ {
		if f.GetSheetName(i) == "data" {
			sheet2Idx = i
			break
		}
	}
	if sheet2Idx < 0 {
		writeError(w, "无效的备份文件：缺少 data 工作表", http.StatusBadRequest)
		return
	}
	sheet2 := "data"

	// Read categories
	catRows, err := f.GetRows(sheet1)
	if err != nil || len(catRows) < 2 {
		writeError(w, "备份文件中分类数据为空", http.StatusBadRequest)
		return
	}

	type catEntry struct {
		Name      string
		SortOrder int
	}
	var cats []catEntry
	for i, row := range catRows {
		if i == 0 {
			continue // skip header
		}
		if len(row) < 2 {
			continue
		}
		name := row[1]
		sortOrder := 0
		if len(row) >= 3 {
			sortOrder, _ = strconv.Atoi(row[2])
		}
		cats = append(cats, catEntry{Name: name, SortOrder: sortOrder})
	}

	// Read data sheet
	dataRows, err := f.GetRows(sheet2)
	if err != nil || len(dataRows) < 1 {
		writeError(w, "备份文件中商品数据为空", http.StatusBadRequest)
		return
	}

	// Parse header to get date columns
	headerRow := dataRows[0]
	log.Printf("Restore DEBUG: header row has %d columns: %v", len(headerRow), headerRow)
	type dateCol struct {
		ColIndex int
		Date     string // normalized to 2006-01-02
	}
	var dateCols []dateCol
	for i := 3; i < len(headerRow); i++ {
		val := headerRow[i]
		for _, layout := range []string{"2006-01-02", "2006/01/02", "01-02-06", "2006-01-02T15:04:05Z", time.RFC3339} {
			if t, err := time.Parse(layout, val); err == nil {
				dateCols = append(dateCols, dateCol{ColIndex: i, Date: t.Format("2006-01-02")})
				break
			}
		}
		if serial, err := strconv.ParseFloat(val, 64); err == nil {
			ds := excelSerialToDate(serial)
			if t, err := time.Parse("2006-01-02", ds); err == nil && t.Year() >= 2020 && t.Year() <= 2035 {
				dateCols = append(dateCols, dateCol{ColIndex: i, Date: ds})
			}
		}
	}
	log.Printf("Restore DEBUG: found %d date columns: %v", len(dateCols), dateCols)

	type priceEntry struct {
		ProductID int
		Date      string
		Price     float64
	}

	type prodEntry struct {
		Name     string
		CatName  string
		Remark   string
		Prices   []priceEntry
	}

	var prods []prodEntry
	for i, row := range dataRows {
		if i == 0 {
			continue // skip header
		}
		if len(row) < 2 {
			continue
		}
		catName := row[0]
		prodName := row[1]
		remark := ""
		if len(row) >= 3 {
			remark = row[2]
		}

		var prices []priceEntry
		for _, dc := range dateCols {
			if dc.ColIndex >= len(row) {
				continue
			}
			val := row[dc.ColIndex]
			price, ok := parseCellPrice(val)
			if !ok {
				continue
			}
			prices = append(prices, priceEntry{
				Date:  dc.Date,
				Price: price,
			})
		}

		prods = append(prods, prodEntry{
			Name:    prodName,
			CatName: catName,
			Remark:  remark,
			Prices:  prices,
		})
	}

	// Execute restore in transaction
	tx, err := h.DB.Begin()
	if err != nil {
		writeError(w, "数据库错误", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Clear tables in order (prices -> products -> categories)
	tx.Exec("DELETE FROM prices")
	tx.Exec("DELETE FROM products")
	tx.Exec("DELETE FROM categories")
	// Reset AUTOINCREMENT counters
	tx.Exec("DELETE FROM sqlite_sequence WHERE name IN ('categories', 'products', 'prices')")

	// Insert categories
	catNameToID := make(map[string]int)
	for _, c := range cats {
		res, err := tx.Exec("INSERT INTO categories (name, sort_order) VALUES (?, ?)", c.Name, c.SortOrder)
		if err != nil {
			writeError(w, "恢复分类失败: "+err.Error(), http.StatusInternalServerError)
			return
		}
		id, _ := res.LastInsertId()
		catNameToID[c.Name] = int(id)
	}

	// Insert products + prices
	totalPrices := 0
	prodNameToID := make(map[string]int)
	for _, p := range prods {
		catID := catNameToID[p.CatName]
		res, err := tx.Exec("INSERT INTO products (name, category_id, remark) VALUES (?, ?, ?)",
			p.Name, catID, p.Remark)
		if err != nil {
			log.Printf("Warning: insert product %s: %v", p.Name, err)
			continue
		}
		pid, _ := res.LastInsertId()
		prodNameToID[p.Name] = int(pid)

		for _, pr := range p.Prices {
			_, err := tx.Exec("INSERT INTO prices (product_id, price_date, price) VALUES (?, ?, ?)",
				pid, pr.Date, pr.Price)
			if err != nil {
				log.Printf("Warning: insert price for %s %s: %v", p.Name, pr.Date, err)
				continue
			}
			totalPrices++
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, "恢复失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Backup restored: %d categories, %d products, %d prices", len(cats), len(prods), totalPrices)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"restored_categories":%d,"restored_products":%d,"restored_prices":%d}`,
		len(cats), len(prods), totalPrices)
}
