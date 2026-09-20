package handler

import "testing"

// TestNormalizePageParams 覆盖会话分页修复：非法/越界参数会被夹到合法范围，
// 保证响应回显的分页元数据与实际使用值一致。
func TestNormalizePageParams(t *testing.T) {
	cases := []struct {
		name         string
		page         string
		pageSize     string
		wantPage     int
		wantPageSize int
	}{
		{"defaults", "1", "20", 1, 20},
		{"zero and negative are clamped", "0", "-3", 1, 20},
		{"non-numeric falls back", "abc", "xyz", 1, 20},
		{"oversized page size falls back", "2", "1000", 2, 20},
		{"valid custom values pass through", "3", "50", 3, 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page, pageSize := normalizePageParams(tc.page, tc.pageSize)
			if page != tc.wantPage || pageSize != tc.wantPageSize {
				t.Fatalf("normalizePageParams(%q,%q) = (%d,%d), want (%d,%d)",
					tc.page, tc.pageSize, page, pageSize, tc.wantPage, tc.wantPageSize)
			}
		})
	}
}
