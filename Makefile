.PHONY: test gen install swagger-ui

SWAGGER_VERSION := v5.17.14
SWAGGER_URL := https://github.com/swagger-api/swagger-ui/archive/refs/tags/$(SWAGGER_VERSION).tar.gz
SWAGGER_TMP_DIR := /tmp/swagger-ui-dist
SWAGGER_DEST_DIR := static/swagger

install:
	go install github.com/arkannsk/elval/cmd/elval-gen@latest

gen: install
	go generate ./...

# Запуск всех тестов
test:
	go test ./... -v

swagger-ui:
	@echo "Downloading Swagger UI $(SWAGGER_VERSION)..."
	@rm -rf $(SWAGGER_TMP_DIR)
	@mkdir -p $(SWAGGER_TMP_DIR)
	@curl -sL $(SWAGGER_URL) | tar xz -C $(SWAGGER_TMP_DIR) --strip-components=1

	@echo "📂 Preparing destination directory..."
	@rm -rf $(SWAGGER_DEST_DIR)
	@mkdir -p $(SWAGGER_DEST_DIR)

	@echo "📦 Copying distribution files..."
	@cp -r $(SWAGGER_TMP_DIR)/dist/* $(SWAGGER_DEST_DIR)/

	@echo "️ Configuring index.html for local spec..."
	@# Заменяем стандартный URL petstore на наш /openapi.json
	@sed -i.bak 's|https://petstore.swagger.io/v2/swagger.json|/openapi.json|g' $(SWAGGER_DEST_DIR)/index.html
	@# Удаляем backup файл, созданный sed
	@rm -f $(SWAGGER_DEST_DIR)/index.html.bak

	@echo "Swagger UI installed successfully in $(SWAGGER_DEST_DIR)"
