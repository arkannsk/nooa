package nooa

import (
	"embed"
	"net/http"
	"strings"
	"time"
)

//go:embed static/swagger/*
var swaggerFS embed.FS

// SwaggerUIHandler возвращает HTTP-хендлер для отдачи Swagger UI.
// Он обслуживает файлы из встроенной файловой системы и перенаправляет запросы на specURL.
func SwaggerUIHandler(specURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Если запрос на корень документации, редиректим на index.html
		if path == "/docs" || path == "/docs/" {
			http.Redirect(w, r, "/docs/index.html", http.StatusFound)
			return
		}

		// Убираем префикс /docs/ для поиска файла в FS
		if strings.HasPrefix(path, "/docs/") {
			filePath := strings.TrimPrefix(path, "/docs/")

			// Пытаемся открыть файл из embedded FS
			file, err := swaggerFS.Open("static/swagger/" + filePath)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer file.Close()

			// Установка Content-Type
			setContentType(w, filePath)

			http.ServeContent(w, r, filePath, time.Now(), file)
			return
		}

		http.NotFound(w, r)
	})
}

// setContentType устанавливает правильный MIME-type для файлов
func setContentType(w http.ResponseWriter, path string) {
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
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
}
