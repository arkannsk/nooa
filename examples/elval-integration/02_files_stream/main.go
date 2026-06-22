package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	files_streams "github.com/arkannsk/nooa/examples/models/02_files_stream"
)

func uploadStandard(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		http.Error(w, "Unable to parse multipart form", http.StatusBadRequest)
		return
	}

	// Создаем структуру для ответа с метаданными
	response := map[string]interface{}{
		"avatar":     "file uploaded",
		"payload":    "file uploaded",
		"thumbnail":  "file uploaded",
		"data":       "data received",
		"attachment": "file uploaded",
	}

	// Просто проверяем, что все файлы загружены успешно
	if _, _, err := r.FormFile("avatar"); err != nil {
		http.Error(w, "Failed to process avatar file", http.StatusBadRequest)
		return
	}

	if _, _, err := r.FormFile("payload"); err != nil {
		http.Error(w, "Failed to process payload file", http.StatusBadRequest)
		return
	}

	if _, _, err := r.FormFile("thumbnail"); err != nil {
		http.Error(w, "Failed to process thumbnail file", http.StatusBadRequest)
		return
	}

	if data := r.FormValue("data"); data == "" {
		http.Error(w, "Data field is required", http.StatusBadRequest)
		return
	}

	if _, _, err := r.FormFile("attachment"); err != nil {
		http.Error(w, "Failed to process attachment file", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func uploadCustom(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		http.Error(w, "Unable to parse multipart form", http.StatusBadRequest)
		return
	}

	// Создаем структуру для ответа с метаданными
	response := map[string]interface{}{
		"avatar":  "custom file uploaded",
		"payload": "custom stream uploaded",
		"data":    "custom buffer uploaded",
	}

	// Просто проверяем, что все файлы загружены успешно
	if _, _, err := r.FormFile("avatar"); err != nil {
		http.Error(w, "Failed to process avatar file", http.StatusBadRequest)
		return
	}

	if _, _, err := r.FormFile("payload"); err != nil {
		http.Error(w, "Failed to process payload file", http.StatusBadRequest)
		return
	}

	if _, _, err := r.FormFile("data"); err != nil {
		http.Error(w, "Failed to process data file", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func uploadMixed(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		http.Error(w, "Unable to parse multipart form", http.StatusBadRequest)
		return
	}

	// Создаем структуру для ответа с метаданными
	response := map[string]interface{}{
		"file":        "file uploaded",
		"userid":      r.FormValue("userid"),
		"description": r.FormValue("description"),
		"tags":        r.Form["tags"],
		"categories":  r.Form["categories"],
	}

	// Проверяем обязательные поля
	if r.FormValue("userid") == "" {
		http.Error(w, "userid field is required", http.StatusBadRequest)
		return
	}

	// Проверяемcategories
	if len(r.Form["categories"]) == 0 {
		http.Error(w, "categories field is required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func main() {
	mux := http.NewServeMux()

	// Создаем спецификацию
	spec := nooa.NewSpec(nooa.Info{
		Title:       "02 Files & Streams Demo",
		Version:     "1.0.0",
		Description: "Integration example: file, stream and byte types mapped to OpenAPI schemas",
	})

	spec.AddTag("FilesStreams", "Файловые типы и потоки в OpenAPI")

	// POST /standard — стандартные файловые типы
	nooa.NewRoute[files_streams.StandardFileTypes, files_streams.StandardFileTypes](
		"POST", "/standard", uploadStandard).
		Summary("Upload with standard file types").
		Description("Demonstrates *os.File, io.Reader, io.ReadCloser, []byte and multipart.File mapped to OpenAPI binary/byte formats").
		Tags("FilesStreams").
		OnSuccess(201, "Standard files uploaded").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	// POST /custom — кастомные типы с аннотациями
	nooa.NewRoute[files_streams.CustomWithAnnotations, files_streams.CustomWithAnnotations](
		"POST", "/custom", uploadCustom).
		Summary("Upload with custom annotated types").
		Description("Demonstrates @oa:file, @oa:stream and @oa:format:byte annotations on custom types").
		Tags("FilesStreams").
		OnSuccess(201, "Custom files uploaded").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	// POST /mixed — смешанный запрос с файлами и обычными полями
	nooa.NewRoute[files_streams.MixedRequest, files_streams.MixedRequest](
		"POST", "/mixed", uploadMixed).
		Summary("Upload with mixed request").
		Description("Demonstrates a request combining file fields with regular string, slice and optional fields").
		Tags("FilesStreams").
		OnSuccess(201, "Mixed upload created").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	// Монтируем документацию
	nooa.RegisterVersionedAPI("", spec, mux)

	log.Println("Server starting on http://localhost:9090")
	log.Println("Swagger UI: http://localhost:9090/docs/")
	log.Println("Raw JSON:   http://localhost:9090/openapi.json")

	log.Fatal(http.ListenAndServe(":9090", mux))
}
