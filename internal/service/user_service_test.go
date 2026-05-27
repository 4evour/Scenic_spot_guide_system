package service

import (
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"too short", "Ab1", true},
		{"no uppercase", "abcdefg1", true},
		{"no lowercase", "ABCDEFG1", true},
		{"no digit", "Abcdefgh", true},
		{"exactly 8 chars valid", "Abcdefg1", false},
		{"long valid", "MyStr0ng!Password", false},
		{"128 chars max", func() string {
			s := "A1bcdefgh"
			for len(s) < 128 {
				s += "x"
			}
			return s
		}(), false},
		{"129 chars too long", func() string {
			s := "A1bcdefgh"
			for len(s) < 129 {
				s += "x"
			}
			return s
		}(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validatePassword(%q) err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}
