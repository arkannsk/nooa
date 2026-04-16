.PHONY: test build generate clean all install

# Переменные
BINARY_NAME=nooa
TESTDATA_DIR=internal/openapi/testdata
OUTPUT_FILE=$(TESTDATA_DIR)/openapi.yaml

# Запуск всех тестов
test:
	go test ./... -v

# Сборка бинарника
build:
	go build -o $(BINARY_NAME) ./cmd/nooa

# Генерация OpenAPI спецификации из testdata
generate: build
	./$(BINARY_NAME) --input $(TESTDATA_DIR) --output $(OUTPUT_FILE) --title "Example API" --version "1.0.0"
	@echo "Generated $(OUTPUT_FILE)"

# Очистка сгенерированных файлов
clean:
	rm -f $(BINARY_NAME)
	rm -f $(OUTPUT_FILE)
	go clean -testcache

# Всё вместе: очистка, тесты, сборка, генерация
all: clean test build generate

# Установка в GOPATH/bin (опционально)
install: build
	go install ./cmd/nooa

# Быстрый запуск тестов без кэша
test-race:
	go test -race ./...

# Помощь
help:
	@echo "Доступные цели:"
	@echo "  test          - запустить все тесты"
	@echo "  test-race     - запустить тесты с детектором гонок"
	@echo "  build         - собрать бинарник nooa"
	@echo "  generate      - сгенерировать openapi.yaml из testdata"
	@echo "  clean         - удалить сгенерированные файлы"
	@echo "  all           - clean + test + build + generate"
	@echo "  install       - установить nooa в GOPATH/bin"
