package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

var ErrInvalidConsumptionScope = errors.New("invalid consumption analysis scope")

type ConsumptionAnalysisService struct {
	path string
}

type ConsumptionAnalysisResponse struct {
	Available bool                 `json:"available"`
	Scope     string               `json:"scope"`
	Period    string               `json:"period"`
	Message   string               `json:"message,omitempty"`
	Analysis  *ConsumptionAnalysis `json:"analysis,omitempty"`
}

type ConsumptionAnalysis struct {
	SchemaVersion     int                         `json:"schema_version"`
	Scope             string                      `json:"scope"`
	SourceMetadata    map[string]any              `json:"source_metadata"`
	Summary           map[string]any              `json:"summary"`
	CategoryBreakdown []map[string]any            `json:"category_breakdown"`
	MonthlyTrend      []map[string]any            `json:"monthly_trend"`
	Segments          map[string][]map[string]any `json:"segments"`
	Recommendations   []string                    `json:"recommendations"`
	DataQuality       map[string]any              `json:"data_quality"`
}

func NewConsumptionAnalysisService(path string) *ConsumptionAnalysisService {
	return &ConsumptionAnalysisService{path: strings.TrimSpace(path)}
}

func (s *ConsumptionAnalysisService) Get(scope, period string) (ConsumptionAnalysisResponse, error) {
	if scope == "" {
		scope = "all"
	}
	if scope != "all" && scope != "lingshan" {
		return ConsumptionAnalysisResponse{}, ErrInvalidConsumptionScope
	}
	if period == "" {
		period = "2025"
	}
	if period != "2025" {
		return ConsumptionAnalysisResponse{}, fmt.Errorf("unsupported consumption analysis period: %s", period)
	}
	response := ConsumptionAnalysisResponse{Scope: scope, Period: period}
	if s == nil || s.path == "" {
		response.Message = "暂无消费分析数据"
		return response, nil
	}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		response.Message = "暂无消费分析数据"
		return response, nil
	}
	if err != nil {
		return ConsumptionAnalysisResponse{}, fmt.Errorf("read consumption analysis: %w", err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		return ConsumptionAnalysisResponse{}, fmt.Errorf("parse consumption analysis: %w", err)
	}
	raw, ok := document[scope]
	if !ok {
		response.Message = "暂无消费分析数据"
		return response, nil
	}
	var analysis ConsumptionAnalysis
	if err := json.Unmarshal(raw, &analysis); err != nil {
		return ConsumptionAnalysisResponse{}, fmt.Errorf("parse consumption analysis scope: %w", err)
	}
	if analysis.Scope == "" {
		analysis.Scope = scope
	}
	response.Available = true
	response.Analysis = &analysis
	return response, nil
}
