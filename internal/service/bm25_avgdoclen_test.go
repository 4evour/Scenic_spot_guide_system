package service

import (
	"math"
	"testing"
)

// TestBM25CalculateScoreHandlesZeroAvgDocLen 覆盖 avgDocLen==0 的除零防护：
// 语料统计异常时不应返回 NaN/Inf 污染检索排序。
func TestBM25CalculateScoreHandlesZeroAvgDocLen(t *testing.T) {
	p := NewBM25FallbackProvider()
	// 构造异常语料状态：有文档但平均长度为 0。
	p.totalDocs = 3
	p.avgDocLen = 0
	p.docFreq = map[string]int{"灵山": 1}

	score := p.CalculateScore([]string{"灵山"}, []string{"灵山", "大佛"})
	if math.IsNaN(score) || math.IsInf(score, 0) {
		t.Fatalf("score = %v, want a finite number", score)
	}
}
