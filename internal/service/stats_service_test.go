package service

import "testing"

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1001, "1,001"},
		{10000, "10,000"},
		{10001, "10,001"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
		{999999999, "999,999,999"},
	}
	for _, tt := range tests {
		got := formatNumber(tt.input)
		if got != tt.want {
			t.Errorf("formatNumber(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
