.PHONY: test gen install swagger-ui

SWAGGER_VERSION := v5.17.14
SWAGGER_URL := https://github.com/swagger-api/swagger-ui/archive/refs/tags/$(SWAGGER_VERSION).tar.gz
SWAGGER_TMP_DIR := /tmp/swagger-ui-dist
SWAGGER_DEST_DIR := static/swagger

REDOC_VERSION := v2.5.3
REDOC_URL := https://cdn.redoc.ly/redoc/$(REDOC_VERSION)/bundles/redoc.standalone.js
REDOC_DEST_DIR := static/redoc

SCALAR_DEST_DIR := static/scalar

install:
	go install github.com/arkannsk/elval/cmd/elval-gen@latest

gen: install
	go generate ./...

gen-spec: install
	elval-gen gen -i ./examples -openapi

# Запуск всех тестов
test:
	go test ./... -v

clean:
	@find ./ -name "*.gen.go" -delete
	@find ./ -name "*.debug.go" -delete

swagger-ui:
	@echo "Downloading Swagger UI $(SWAGGER_VERSION)..."
	@rm -rf $(SWAGGER_TMP_DIR)
	@mkdir -p $(SWAGGER_TMP_DIR)
	@curl -sL $(SWAGGER_URL) | tar xz -C $(SWAGGER_TMP_DIR) --strip-components=1

	@echo "Preparing destination directory..."
	@rm -rf $(SWAGGER_DEST_DIR)
	@mkdir -p $(SWAGGER_DEST_DIR)

	@echo "Copying distribution files..."
	@cp -r $(SWAGGER_TMP_DIR)/dist/* $(SWAGGER_DEST_DIR)/

	@echo "️  Patching configuration..."
	@if [ -f "$(SWAGGER_DEST_DIR)/swagger-initializer.js" ]; then \
		echo "   Found swagger-initializer.js, patching..."; \
		sed -i.bak 's|https://petstore.swagger.io/v2/swagger.json|/openapi.json|g' $(SWAGGER_DEST_DIR)/swagger-initializer.js; \
		rm -f $(SWAGGER_DEST_DIR)/swagger-initializer.js.bak; \
	else \
		echo "   swagger-initializer.js not found, trying index.html..."; \
		sed -i.bak 's|https://petstore.swagger.io/v2/swagger.json|/openapi.json|g' $(SWAGGER_DEST_DIR)/index.html; \
		rm -f $(SWAGGER_DEST_DIR)/index.html.bak; \
	fi

	@echo "Swagger UI installed and patched successfully!"
	@echo "IMPORTANT: Restart your Go server to embed changes."

redoc:
	@echo "Downloading Redoc $(REDOC_VERSION)..."
	@mkdir -p $(REDOC_DEST_DIR)
	@curl -sL $(REDOC_URL) -o $(REDOC_DEST_DIR)/redoc.standalone.js
	@printf '<!DOCTYPE html>\n<html>\n<head>\n  <meta charset="UTF-8">\n  <title>API Documentation</title>\n  <style>\n    body { margin: 0; padding: 0; }\n  </style>\n</head>\n<body>\n  <redoc spec-url="{{SPEC_URL}}"></redoc>\n  <script src="./redoc.standalone.js"></script>\n</body>\n</html>\n' > $(REDOC_DEST_DIR)/index.html
	@echo "Redoc installed successfully!"
	@echo "IMPORTANT: Restart your Go server to embed changes."

scalar:
	@echo "Downloading Scalar..."
	@mkdir -p $(SCALAR_DEST_DIR)
	@curl -sL https://cdn.jsdelivr.net/npm/@scalar/api-reference@latest/dist/browser/standalone.js -o $(SCALAR_DEST_DIR)/scalar.min.js
	@printf '<!DOCTYPE html>\n<html>\n<head>\n  <meta charset="UTF-8">\n  <title>API Documentation</title>\n  <style>\n    body { margin: 0; padding: 0; }\n  </style>\n</head>\n<body>\n  <script id="api-reference" type="application/json" data-url="{{SPEC_URL}}"></script>\n  <script src="./scalar.min.js"></script>\n</body>\n</html>\n' > $(SCALAR_DEST_DIR)/index.html
	@echo "Scalar installed successfully!"
	@echo "IMPORTANT: Restart your Go server to embed changes."
