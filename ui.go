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

func SwaggerUIHandler(basePrefix string, specURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		cleanPath := strings.TrimSuffix(path, "/")

		// Редирект на index.html если пришел на корень префикса
		if cleanPath == basePrefix || path == basePrefix+"/" {
			http.Redirect(w, r, path+"index.html", http.StatusFound)
			return
		}

		// Проверка префикса
		if !strings.HasPrefix(path, basePrefix+"/") && path != basePrefix {
			http.NotFound(w, r)
			return
		}

		// Извлекаем имя файла относительно префикса
		filePath := strings.TrimPrefix(path, basePrefix+"/")

		if filePath == "" {
			filePath = "index.html"
		}

		// Открываем файл из embedded FS
		f, err := swaggerFS.Open("static/swagger/" + filePath)
		if err != nil {
			log.Printf("❌ ERROR: File not found in embed: static/swagger/%s. Err: %v", filePath, err)
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		stat, err := f.Stat()
		isModifiedFile := (filePath == "index.html" || filePath == "swagger-initializer.js")

		// Устанавливаем заголовки только если файл НЕ будет модифицирован
		// Или если мы готовы пересчитать длину (что сложнее)
		if stat != nil && !isModifiedFile {
			w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
			if !stat.ModTime().IsZero() {
				w.Header().Set("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))
			}
		}

		// Устанавливаем Content-Type ДО записи любого контента
		setContentType(w, filePath)

		// Обрабатываем файлы, где нужно подменять URL
		if isModifiedFile {
			data, err := io.ReadAll(f)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			content := strings.Replace(string(data), "{{SPEC_URL}}", specURL, -1)

			// Пишем данные без Content-Length, так как размер изменился
			_, err = w.Write([]byte(content))
			if err != nil {
				log.Printf("Error writing response for %s: %v", filePath, err)
			}
			return
		}

		// Для остальных файлов просто копируем контент
		_, err = io.Copy(w, f)
		if err != nil {
			log.Printf("Error copying %s to response: %v", filePath, err)
		}
	})
}

func setContentType(w http.ResponseWriter, path string) {
	switch {
	case strings.HasSuffix(path, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(path, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case strings.HasSuffix(path, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case strings.HasSuffix(path, ".png"):
		w.Header().Set("Content-Type", "image/png")
	case strings.HasSuffix(path, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	case strings.HasSuffix(path, ".json"):
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
}
