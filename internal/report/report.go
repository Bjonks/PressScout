package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"pressscout/internal/model"
)

type Summary struct {
	Total    int                          `json:"total"`
	Internal int                          `json:"internal"`
	External int                          `json:"external"`
	ByClass  map[model.Classification]int `json:"by_classification"`
}

func Summarize(results []model.Result) Summary {
	s := Summary{ByClass: make(map[model.Classification]int), Total: len(results)}
	for _, result := range results {
		if result.External {
			s.External++
		} else {
			s.Internal++
		}
		s.ByClass[result.Class]++
	}
	return s
}

func PrintText(w io.Writer, results []model.Result) error {
	s := Summarize(results)
	if _, err := fmt.Fprintf(w, "Checked: %d (internal: %d, external: %d)\n", s.Total, s.Internal, s.External); err != nil {
		return err
	}
	classes := make([]string, 0, len(s.ByClass))
	for class := range s.ByClass {
		classes = append(classes, string(class))
	}
	sort.Strings(classes)
	for _, class := range classes {
		if _, err := fmt.Fprintf(w, "%s: %d\n", class, s.ByClass[model.Classification(class)]); err != nil {
			return err
		}
	}
	for _, result := range results {
		if result.Class == model.OK {
			continue
		}
		if _, err := fmt.Fprintf(w, "\n%s [%s]", result.OriginalURL, result.Class); err != nil {
			return err
		}
		if result.FinalURL != "" && result.FinalURL != result.OriginalURL {
			if _, err := fmt.Fprintf(w, " -> %s", result.FinalURL); err != nil {
				return err
			}
		}
		if result.Status != 0 {
			if _, err := fmt.Fprintf(w, " status=%d", result.Status); err != nil {
				return err
			}
		}
		if result.Error != "" {
			if _, err := fmt.Fprintf(w, " error=%s", result.Error); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		for _, source := range result.Sources {
			if _, err := fmt.Fprintf(w, "  source: %s\n", source); err != nil {
				return err
			}
		}
	}
	return nil
}

type JSONReport struct {
	Summary Summary        `json:"summary"`
	Results []model.Result `json:"results"`
}

func WriteJSON(filename string, results []model.Result) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create JSON output: %w", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(JSONReport{Summary: Summarize(results), Results: results}); err != nil {
		return fmt.Errorf("write JSON output: %w", err)
	}
	return nil
}
