package client

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// --- CSVCodec tests ---

func TestCSVCodec_Unmarshal(t *testing.T) {
	csvData := `name,age,city
Alice,30,NYC
Bob,25,LA`

	var result [][]string
	if err := (&CSVCodec{}).Unmarshal([]byte(csvData), &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(result))
	}
	if result[0][0] != "name" || result[1][0] != "Alice" {
		t.Fatalf("unexpected data: %v", result)
	}
}

func TestCSVCodec_Unmarshal_Quoted(t *testing.T) {
	csvData := `name,desc
"Alice","has a ""quote"" here"
"Bob","simple"`

	var result [][]string
	if err := (&CSVCodec{}).Unmarshal([]byte(csvData), &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 rows (header + 2 data), got %d", len(result))
	}
	if result[1][1] != `has a "quote" here` {
		t.Fatalf("expected escaped quote, got %q", result[1][1])
	}
}

func TestCSVCodec_Unmarshal_Empty(t *testing.T) {
	var result [][]string
	if err := (&CSVCodec{}).Unmarshal([]byte(""), &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(result))
	}
}

func TestCSVCodec_Unmarshal_WrongTarget(t *testing.T) {
	var result string
	err := (&CSVCodec{}).Unmarshal([]byte("a,b"), &result)
	if err == nil {
		t.Fatal("expected error: target must be *[][]string")
	}
}

func TestCSVCodec_Marshal_NoHeader(t *testing.T) {
	data := [][]string{
		{"name", "age"},
		{"Alice", "30"},
		{"Bob", "25"},
	}

	b, err := (&CSVCodec{}).Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result [][]string
	if err := (&CSVCodec{}).Unmarshal(b, &result); err != nil {
		t.Fatalf("Unmarshal roundtrip failed: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(result))
	}
	if result[0][0] != "name" || result[1][0] != "Alice" {
		t.Fatalf("roundtrip mismatch: %v", result)
	}
}

func TestCSVCodec_Marshal_WrongType(t *testing.T) {
	_, err := (&CSVCodec{}).Marshal("not csv")
	if err == nil {
		t.Fatal("expected error for non-[][]string type")
	}
}

func TestUnmarshalBody_CSV(t *testing.T) {
	csvData := `id,value
1,hello
2,world`

	var result [][]string
	if err := UnmarshalBody([]byte(csvData), "text/csv", nil, &result); err != nil {
		t.Fatalf("UnmarshalBody failed: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(result))
	}
	if result[2][1] != "world" {
		t.Fatalf("expected 'world', got %q", result[2][1])
	}
}

func TestUnmarshalResponse_CSV(t *testing.T) {
	csvData := `a,b,c
1,2,3`

	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(csvData)),
		Header: map[string][]string{
			"Content-Type": {"text/csv"},
		},
	}

	var result [][]string
	if err := UnmarshalResponse(resp, nil, &result); err != nil {
		t.Fatalf("UnmarshalResponse failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}
}

// --- Roundtrip ---

func TestCSV_Roundtrip(t *testing.T) {
	original := [][]string{
		{"col1", "col2", "col3"},
		{"val1", "val2", "val3"},
		{"has,comma", "has\nnewline", `has"quote`},
	}

	b, err := (&CSVCodec{}).Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result [][]string
	if err := (&CSVCodec{}).Unmarshal(b, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(result) != len(original) {
		t.Fatalf("row count mismatch: %d vs %d", len(result), len(original))
	}
	for i := range original {
		if len(result[i]) != len(original[i]) {
			t.Fatalf("col count mismatch at row %d", i)
		}
		for j := range original[i] {
			if result[i][j] != original[i][j] {
				t.Fatalf("mismatch at [%d][%d]: %q vs %q", i, j, result[i][j], original[i][j])
			}
		}
	}
}
