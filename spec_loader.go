package nooa

import (
	"encoding/json"
	"os"
)

// LoadSchemasFromFile загружает JSON со схемами из файла (например, сгенерированный elval)
func LoadSchemasFromFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var schemas map[string]any
	if err := json.Unmarshal(data, &schemas); err != nil {
		return nil, err
	}
	return schemas, nil
}
