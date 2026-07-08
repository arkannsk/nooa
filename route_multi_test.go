package nooa

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type CreatedRes struct {
	ID string `json:"id"`
}

type AcceptedRes struct {
	JobID string `json:"job_id"`
}

func TestNewRouteMultiResp(t *testing.T) {
	ClearRegistry()
	defer ClearRegistry()

	mux := http.NewServeMux()

	b := NewRouteMultiResp[Req]("POST", "/jobs", handlerOK,
		ResponseEntry{Status: 201, Instance: new(CreatedRes), Desc: "Created"},
		ResponseEntry{Status: 202, Instance: new(AcceptedRes), Desc: "Accepted"},
	).Summary("Create job").Tags("jobs")

	spec := b.Spec()

	if spec.Method != "POST" || spec.Path != "/jobs" {
		t.Fatalf("unexpected method/path: %s %s", spec.Method, spec.Path)
	}

	// Должны быть 2 ответа
	if len(spec.Responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(spec.Responses))
	}

	// Проверка статусов
	statuses := make(map[int]bool)
	for _, r := range spec.Responses {
		statuses[r.Status] = true
	}
	if !statuses[201] || !statuses[202] {
		t.Fatalf("expected statuses 201 and 202, got %v", statuses)
	}

	// Проверка схем
	schemaNames := spec.ResponseSchemaNames
	if _, ok := schemaNames[201]; !ok {
		t.Error("missing schema for 201")
	}
	if _, ok := schemaNames[202]; !ok {
		t.Error("missing schema for 202")
	}
	// Схемы для разных статусов должны быть разными
	if schemaNames[201] == schemaNames[202] {
		t.Error("schemas for 201 and 202 should be different")
	}

	// RegisterSpecAndMux с nil spec
	b.RegisterSpecAndMux(mux, nil)

	// Проверка, что роут зарегистрирован в mux
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/jobs", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from mux, got %d", rec.Code)
	}
}
