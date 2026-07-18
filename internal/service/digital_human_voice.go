package service

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const DefaultDigitalHumanVoiceID = "female_xiaoxiao"

var digitalHumanVoiceIDs = map[string]bool{
	"female_xiaoxiao": true,
	"female_xiaoyi":   true,
	"female_yunxi":    true,
	"male_yunyang":    true,
	"male_yunjian":    true,
	"female_xiaobei":  true,
	"female_yunxia":   true,
}

var digitalHumanVoiceByType = map[string]string{
	"温柔自然女声": DefaultDigitalHumanVoiceID,
	"沉稳专业女声": "female_xiaoyi",
	"活力亲切女声": "female_yunxi",
	"端庄礼仪女声": DefaultDigitalHumanVoiceID,
}

var ttsRatePattern = regexp.MustCompile(`^[+-](?:[0-9]|[1-9][0-9]|100)%$`)

func IsValidDigitalHumanVoiceID(id string) bool {
	return digitalHumanVoiceIDs[id]
}

func NormalizeDigitalHumanVoiceID(id, voiceType string) string {
	if IsValidDigitalHumanVoiceID(id) {
		return id
	}
	if mapped, ok := digitalHumanVoiceByType[voiceType]; ok {
		return mapped
	}
	return DefaultDigitalHumanVoiceID
}

func ValidateDigitalHumanVoiceID(id string) error {
	if id == "" || IsValidDigitalHumanVoiceID(id) {
		return nil
	}
	return fmt.Errorf("unknown digital human voice: %s", id)
}

func NormalizeDigitalHumanSpeed(speed float64) float64 {
	if speed <= 0 {
		return 0.8
	}
	if speed < 0.5 {
		return 0.5
	}
	if speed > 2.0 {
		return 2.0
	}
	return speed
}

func NormalizeDigitalHumanTTSRate(rate string, speed float64) string {
	rate = strings.TrimSpace(rate)
	if ttsRatePattern.MatchString(rate) {
		return rate
	}
	speed = NormalizeDigitalHumanSpeed(speed)
	percent := int(math.Round((speed - 1.0) * 100))
	if percent > 100 {
		percent = 100
	}
	if percent < -50 {
		percent = -50
	}
	if percent >= 0 {
		return "+" + strconv.Itoa(percent) + "%"
	}
	return strconv.Itoa(percent) + "%"
}

func ValidateDigitalHumanSettings(speed float64, volume, emotionLevel int, emotion, voiceID string) error {
	if speed < 0.5 || speed > 2.0 {
		return fmt.Errorf("digital human speed must be between 0.5 and 2.0")
	}
	if volume < 0 || volume > 100 {
		return fmt.Errorf("digital human volume must be between 0 and 100")
	}
	if emotionLevel < 1 || emotionLevel > 5 {
		return fmt.Errorf("digital human emotion level must be between 1 and 5")
	}
	if emotion != "neutral" && emotion != "joy" && emotion != "surprise" && emotion != "sadness" && emotion != "anger" && emotion != "fear" && emotion != "disgust" {
		return fmt.Errorf("unknown digital human emotion: %s", emotion)
	}
	return ValidateDigitalHumanVoiceID(voiceID)
}
