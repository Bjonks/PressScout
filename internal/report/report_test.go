package report

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"wpgopher/internal/model"
)

func TestPrintTextAndWriteJSON(t *testing.T) {
	results := []model.Result{{OriginalURL: "https://example.test/bad", Class: model.Broken, Status: 404, Sources: []string{"https://example.test/"}}}
	var text bytes.Buffer
	if err := PrintText(&text, results); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text.String(), "BROKEN") || !strings.Contains(text.String(), "source:") {
		t.Fatalf("text = %s", text.String())
	}
	file := t.TempDir() + "/results.json"
	if err := WriteJSON(file, results); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(file)
	var decoded JSONReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Summary.Total != 1 || len(decoded.Results) != 1 {
		t.Fatalf("decoded = %+v", decoded)
	}
}
