package nooa

import (
	"embed"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

//go:embed static/swagger/*
var swaggerFS embed.FS

// SwaggerUIHandler возвращает HTTP-хендлер для отдачи Swagger UI.
func SwaggerUIHandler(specURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Редирект с /docs на /docs/index.html
		if path == "/docs" || path == "/docs/" {
			http.Redirect(w, r, "/docs/index.html", http.StatusFound)
			return
		}

		// Обработка статических файлов
		if strings.HasPrefix(path, "/docs/") {
			filePath := strings.TrimPrefix(path, "/docs/")

			// Открываем файл из embedded FS
			f, err := swaggerFS.Open("static/swagger/" + filePath)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer f.Close()

			// Пытаемся получить stat для заголовков (размер, время)
			stat, err := f.Stat()
			if err == nil {
				w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
				// Если есть время модификации, ставим его для кэширования
				if !stat.ModTime().IsZero() {
					w.Header().Set("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))
				}
			}

			// Установка Content-Type
			setContentType(w, filePath)

			_, err = io.Copy(w, f)
			if err != nil {
				log.Printf("Error copying %s to http.ServeMux: %v", path, err)
				return
			}
			return
		}

		http.NotFound(w, r)
	})
}

func setContentType(w http.ResponseWriter, path string) {
	// Простая эвристика для Content-Type
	// В продакшене лучше использовать пакет mime.TypeByExtension
	switch {
	case strings.HasSuffix(path, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(path, ".js"):
		w.Header().Set("Content-Type", "application/javascript")
	case strings.HasSuffix(path, ".css"):
		w.Header().Set("Content-Type", "text/css")
	case strings.HasSuffix(path, ".png"):
		w.Header().Set("Content-Type", "image/png")
	case strings.HasSuffix(path, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	case strings.HasSuffix(path, ".json"):
		w.Header().Set("Content-Type", "application/json")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
}
