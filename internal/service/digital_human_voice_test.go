package service

import "testing"

func TestNormalizeDigitalHumanVoiceIDUsesFriendlyVoiceType(t *testing.T) {
	if got := NormalizeDigitalHumanVoiceID("", "沉稳专业女声"); got != "female_xiaoyi" {
		t.Fatalf("voice = %q, want female_xiaoyi", got)
	}
	if got := NormalizeDigitalHumanVoiceID("male_yunjian", "温柔自然女声"); got != "male_yunjian" {
		t.Fatalf("explicit voice = %q, want male_yunjian", got)
	}
}

func TestNormalizeDigitalHumanTTSRate(t *testing.T) {
	if got := NormalizeDigitalHumanTTSRate("", 0.8); got != "-20%" {
		t.Fatalf("rate = %q, want -20%%", got)
	}
	if got := NormalizeDigitalHumanTTSRate("+15%", 0.8); got != "+15%" {
		t.Fatalf("explicit rate = %q, want +15%%", got)
	}
	if got := NormalizeDigitalHumanTTSRate("invalid", 2.5); got != "+100%" {
		t.Fatalf("clamped rate = %q, want +100%%", got)
	}
}

func TestValidateDigitalHumanSettings(t *testing.T) {
	if err := ValidateDigitalHumanSettings(0.8, 80, 3, "joy", "female_xiaoxiao"); err != nil {
		t.Fatalf("valid settings rejected: %v", err)
	}
	if err := ValidateDigitalHumanSettings(2.1, 80, 3, "joy", "female_xiaoxiao"); err == nil {
		t.Fatal("expected speed validation error")
	}
	if err := ValidateDigitalHumanSettings(0.8, 80, 3, "joy", "unknown"); err == nil {
		t.Fatal("expected voice validation error")
	}
}
