package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"github.com/mhxy/price-tracker/database"
	"github.com/mhxy/price-tracker/handlers"
	"github.com/mhxy/price-tracker/middleware"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	port := flag.String("port", "8081", "server port")
	dbPath := flag.String("db", "data/mhxy.db", "database path")
	ocrEndpoint := flag.String("ocr-endpoint", "http://192.168.0.105:7890/v1/chat/completions", "OCR AI model endpoint")
	ocrModel := flag.String("ocr-model", "glm-ocr", "OCR AI model name")
	initDB := flag.Bool("init-db", false, "initialize database and seed data")
	xlsxPath := flag.String("xlsx", "", "path to xlsx file for seeding")
	flag.Parse()

	if err := database.InitDB(*dbPath, map[string]string{
		"ocr_endpoint": *ocrEndpoint,
		"ocr_model":    *ocrModel,
	}); err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	log.Printf("Database initialized: %s", *dbPath)

	if *initDB {
		if err := database.SeedData(*xlsxPath); err != nil {
			log.Fatalf("Failed to seed data: %v", err)
		}
		log.Println("Database seeded successfully!")
	}

	mux := http.NewServeMux()

	authH := &handlers.AuthHandler{DB: database.DB}
	productH := &handlers.ProductHandler{DB: database.DB}
	priceH := &handlers.PriceHandler{DB: database.DB}
	categoryH := &handlers.CategoryHandler{DB: database.DB}
	ocrH := &handlers.OcrHandler{Endpoint: *ocrEndpoint, Model: *ocrModel, DB: database.DB}
	settingsH := &handlers.SettingsHandler{DB: database.DB}

	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			authH.Login(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/categories", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			categoryH.List(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	productH.RegisterRoutes(mux)
	priceH.RegisterRoutes(mux)

	// OCR endpoint
	mux.HandleFunc("/api/ocr", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			middleware.AuthMiddleware(ocrH.Recognize)(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	// Batch price endpoint
	mux.HandleFunc("/api/prices/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			middleware.AuthMiddleware(priceH.BatchCreate)(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	// Settings endpoints
	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			middleware.AuthMiddleware(settingsH.GetSettings)(w, r)
		case http.MethodPut:
			middleware.AuthMiddleware(settingsH.UpdateSettings)(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/settings/password", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			middleware.AuthMiddleware(settingsH.ChangePassword)(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/trend", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			priceH.Trend(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	// Serve static files (embedded frontend)
	staticSub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Printf("Warning: static files not found, API-only mode")
	} else {
		fileServer := http.FileServer(http.FS(staticSub))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}
			if _, err := fs.Stat(staticSub, path); err != nil {
				r.URL.Path = "/"
			}
			// Disable cache for index.html, allow cache for hashed assets
			if path == "index.html" {
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	// Recovery + CORS + logging middleware
	handler := recoveryMiddleware(corsMiddleware(loggingMiddleware(mux)))

	addr := fmt.Sprintf("0.0.0.0:%s", *port)
	log.Printf("Server starting on http://%s", addr)
	log.Printf("OCR model: %s @ %s", *ocrModel, *ocrEndpoint)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %v", err)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
