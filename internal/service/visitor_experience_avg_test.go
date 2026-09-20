package service

import "testing"

// TestAvgOrZero 覆盖评分维度均值修复：可选维度的 0（未评分）不应计入分母。
func TestAvgOrZero(t *testing.T) {
	cases := []struct {
		name string
		sum  int
		n    int
		want float64
	}{
		{"no ratings", 0, 0, 0},
		{"single 5", 5, 1, 5},
		{"only the rated dimension counts", 5, 1, 5}, // overall=5 + culture=0(未评) 时，culture 只按 1 条算
		{"two fives", 10, 2, 5},
		{"rounds to one decimal", 10, 3, 3.3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := avgOrZero(tc.sum, tc.n); got != tc.want {
				t.Fatalf("avgOrZero(%d,%d) = %v, want %v", tc.sum, tc.n, got, tc.want)
			}
		})
	}
}
