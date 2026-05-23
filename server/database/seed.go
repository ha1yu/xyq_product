package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
)

var categories = []struct {
	Name      string
	SortOrder int
}{
	{"消耗品", 1},
	{"炼妖石", 2},
	{"图册", 3},
	{"如意丹", 4},
	{"宝石", 5},
	{"装备石", 6},
	{"珍珠", 7},
	{"书铁", 8},
	{"元宵", 9},
	{"附魔宝珠", 10},
	{"兽诀", 11},
	{"阵法", 12},
	{"其他", 13},
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
	if strings.HasPrefix(name, "炼妖石") {
		return "炼妖石"
	}
	if strings.HasPrefix(name, "图册") {
		return "图册"
	}
	if strings.HasPrefix(name, "如意丹") {
		return "如意丹"
	}
	if name == "金刚石" || name == "定魂珠" || name == "夜光珠" || name == "龙鳞" || name == "避水珠" {
		return "宝石"
	}
	if name == "太阳石" || name == "舍利子" || name == "月亮石" ||
		name == "黑宝石" || name == "红玛瑙" || name == "光芒石" {
		return "装备石"
	}
	if strings.HasPrefix(name, "珍珠") {
		return "珍珠"
	}
	if strings.HasPrefix(name, "书") || strings.HasPrefix(name, "铁") {
		return "书铁"
	}
	if name == "战魄" {
		return "书铁"
	}
	if strings.HasSuffix(name, "元宵") {
		return "元宵"
	}
	if strings.HasPrefix(name, "附魔宝珠") {
		return "附魔宝珠"
	}
	if strings.HasSuffix(name, "阵") {
		return "阵法"
	}
	if strings.HasPrefix(name, "种子") {
		return "消耗品"
	}
	if name == "仙露小丸子" {
		return "其他"
	}
	if strings.Contains(name, "内丹") || name == "精岳" || name == "矫健" ||
		name == "迅敏" || name == "双星爆" || name == "腾挪劲" {
		return "其他"
	}
	return "兽诀"
}

// Valid Excel serial date numbers (skip 260308)
var validSerialDates = map[float64]bool{
	44197: true, 44507: true, 44686: true, 44736: true,
	45817: true, 45857: true, 45907: true,
}

func excelSerialToDate(serial float64) string {
	// Excel epoch: 1899-12-30 is serial 0
	baseTime, _ := time.Parse("2006-01-02", "1899-12-30")
	result := baseTime.AddDate(0, 0, int(serial))
	return result.Format("2006-01-02")
}

var priceRegex = regexp.MustCompile(`[\d.]+`)

func parseCellPrice(cellVal string) (float64, string, bool) {
	cellVal = strings.TrimSpace(cellVal)
	if cellVal == "" {
		return 0, "", false
	}
	if num, err := strconv.ParseFloat(cellVal, 64); err == nil && num > 0 {
		return num, "", true
	}
	numStr := priceRegex.FindString(cellVal)
	if numStr == "" {
		return 0, "", false
	}
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, "", false
	}
	remark := ""
	if cleaned := strings.TrimSpace(cellVal); cleaned != numStr {
		remark = cleaned
	}
	return num, remark, true
}

func SeedData(xlsxPath string) error {
	// Seed admin
	var adminCount int
	DB.QueryRow("SELECT COUNT(*) FROM admins").Scan(&adminCount)
	if adminCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		_, err = DB.Exec("INSERT INTO admins (username, password) VALUES (?, ?)", "admin", string(hash))
		if err != nil {
			return fmt.Errorf("insert admin: %w", err)
		}
		log.Println("Created admin user: admin/123456")
	}

	// Seed categories
	catIDMap := make(map[string]int)
	for _, cat := range categories {
		var id int
		err := DB.QueryRow("SELECT id FROM categories WHERE name = ?", cat.Name).Scan(&id)
		if err == sql.ErrNoRows {
			res, err := DB.Exec("INSERT INTO categories (name, sort_order) VALUES (?, ?)", cat.Name, cat.SortOrder)
			if err != nil {
				return fmt.Errorf("insert category %s: %w", cat.Name, err)
			}
			lastID, _ := res.LastInsertId()
			id = int(lastID)
		}
		catIDMap[cat.Name] = id
	}

	// Check if products already seeded
	var prodCount int
	DB.QueryRow("SELECT COUNT(*) FROM products").Scan(&prodCount)
	if prodCount > 0 {
		log.Println("Data already seeded, skipping...")
		return nil
	}

	if xlsxPath == "" {
		xlsxPath = findXlsx()
	}
	if xlsxPath == "" {
		return fmt.Errorf("xlsx file not found")
	}

	f, err := excelize.OpenFile(xlsxPath)
	if err != nil {
		return fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return fmt.Errorf("read sheet: %w", err)
	}
	if len(rows) == 0 {
		return fmt.Errorf("empty sheet")
	}

	// Parse header - excelize reads date cells as numeric strings
	headerRow := rows[0]
	type dateCol struct {
		ColIndex int
		Date     string
	}
	var dateCols []dateCol
	for i, cell := range headerRow {
		if i == 0 {
			continue // skip "商品名称"
		}
		serial, err := strconv.ParseFloat(cell, 64)
		if err != nil {
			continue
		}
		if !validSerialDates[serial] {
			log.Printf("Skipping invalid date column %d: serial=%.0f\n", i, serial)
			continue
		}
		dateStr := excelSerialToDate(serial)
		dateCols = append(dateCols, dateCol{ColIndex: i, Date: dateStr})
		log.Printf("Date column %d: serial=%.0f -> %s\n", i, serial, dateStr)
	}

	log.Printf("Found %d valid date columns\n", len(dateCols))

	// Insert data in transaction
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmtProd, err := tx.Prepare("INSERT INTO products (name, category_id, remark) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtProd.Close()

	stmtPrice, err := tx.Prepare("INSERT OR IGNORE INTO prices (product_id, price_date, price) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtPrice.Close()

	inserted := 0
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

		res, err := stmtProd.Exec(name, catID, "")
		if err != nil {
			log.Printf("Warning: insert product %s: %v\n", name, err)
			continue
		}
		productID, _ := res.LastInsertId()
		inserted++

		var remarks []string
		for _, dc := range dateCols {
			if dc.ColIndex >= len(row) {
				continue
			}
			cellVal := row[dc.ColIndex]
			if cellVal == "" {
				continue
			}
			priceVal, remark, ok := parseCellPrice(cellVal)
			if !ok {
				continue
			}
			_, err := stmtPrice.Exec(productID, dc.Date, priceVal)
			if err != nil {
				log.Printf("Warning: insert price %s %s: %v\n", name, dc.Date, err)
			}
			if remark != "" {
				remarks = append(remarks, fmt.Sprintf("%s: %s", dc.Date, remark))
			}
		}
		if len(remarks) > 0 {
			tx.Exec("UPDATE products SET remark = ? WHERE id = ?", strings.Join(remarks, "; "), productID)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	log.Printf("Data seeded successfully! %d products inserted.\n", inserted)
	return nil
}

func findXlsx() string {
	candidates := []string{
		"梦幻将军物价表.xlsx",
		"../梦幻将军物价表.xlsx",
		"../../梦幻将军物价表.xlsx",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}
