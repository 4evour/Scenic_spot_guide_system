package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConsumptionAnalysisServiceLoadsScopeAndReturnsNoDataHonestly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "consumption.json")
	if err := os.WriteFile(path, []byte(`{
  "all": {
    "schema_version": 1,
    "scope": "all",
    "source_metadata": {"source_row_count": 2},
    "summary": {"sample_count": 2, "total_cost": 100},
    "category_breakdown": [],
    "monthly_trend": [],
    "segments": {},
    "recommendations": ["建议"],
    "data_quality": {"invalid_numeric_rows": 0}
  }
}`), 0o600); err != nil {
		t.Fatalf("write analysis: %v", err)
	}

	service := NewConsumptionAnalysisService(path)
	result, err := service.Get("all", "2025")
	if err != nil {
		t.Fatalf("get analysis: %v", err)
	}
	if !result.Available || result.Analysis == nil || result.Analysis.Summary["sample_count"] != float64(2) {
		t.Fatalf("unexpected analysis: %+v", result)
	}

	missing, err := NewConsumptionAnalysisService(filepath.Join(dir, "missing.json")).Get("lingshan", "2025")
	if err != nil {
		t.Fatalf("missing analysis should not error: %v", err)
	}
	if missing.Available || missing.Message == "" {
		t.Fatalf("missing analysis should be explicit: %+v", missing)
	}
}

func TestConsumptionAnalysisServiceRejectsInvalidScopeAndPeriod(t *testing.T) {
	service := NewConsumptionAnalysisService("")
	if _, err := service.Get("other", "2025"); err != ErrInvalidConsumptionScope {
		t.Fatalf("scope error = %v", err)
	}
	if _, err := service.Get("all", "2024"); err == nil {
		t.Fatal("expected unsupported period error")
	}
}
