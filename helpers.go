package nooa

import (
	"net/http"
	"strings"
)

// RegisterVersionedAPI регистрирует изолированную спецификацию API для указанной версии.
// versionPrefix: префикс версии (например "v1"). Если пустая строка "", используется корень.
func RegisterVersionedAPI(versionPrefix string, spec *Spec, mux *http.ServeMux) {
	if spec == nil || mux == nil {
		return
	}

	prefix := strings.Trim(versionPrefix, "/")

	var jsonPath, docsBase string

	if prefix == "" {
		// Для корня API
		jsonPath = "/openapi.json"
		docsBase = "/docs"
	} else {
		// Для версий
		// JSON лежит по пути /{version}/openapi.json
		jsonPath = "/" + prefix + "/openapi.json"

		// Swagger UI лежит по пути /docs/{version}/
		docsBase = "/docs/" + prefix
	}

	// 1. Монтируем отдачу OpenAPI JSON
	mux.Handle(jsonPath, spec)

	// 2. Монтируем Swagger UI
	// SwaggerUIHandler ожидает basePrefix, который будет урезаться из пути запроса
	// Например, если запрос /docs/v1/index.html, то basePrefix="/docs/v1"
	swaggerHandler := SwaggerUIHandler(docsBase, jsonPath)

	// Регистрируем хендлер для всех подпутей /docs/v1/...
	mux.Handle(docsBase+"/", swaggerHandler)

	// Регистрируем редирект для точного пути /docs/v1 (без слэша)
	mux.HandleFunc(docsBase, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, docsBase+"/", http.StatusMovedPermanently)
	})
}
