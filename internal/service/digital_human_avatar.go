package service

import "fmt"

const DefaultDigitalHumanAvatarID = "mao_pro"

type DigitalHumanAvatarOption struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	ModelURL      string `json:"model_url"`
	ConfigFile    string `json:"config_file"`
	PreviewColor  string `json:"preview_color"`
	FallbackLabel string `json:"fallback_label"`
}

var digitalHumanAvatarOptions = []DigitalHumanAvatarOption{
	{
		ID:            "mao_pro",
		Name:          "Niziiro Mao（临时示例）",
		Description:   "Live2D 官方 PRO 示例模型",
		ModelURL:      "/static/live2d-models/mao_pro/runtime/mao_pro.model3.json",
		ConfigFile:    "conf.yaml",
		PreviewColor:  "#D4AF37",
		FallbackLabel: "M",
	},
}

func DigitalHumanAvatarOptions() []DigitalHumanAvatarOption {
	options := make([]DigitalHumanAvatarOption, len(digitalHumanAvatarOptions))
	copy(options, digitalHumanAvatarOptions)
	return options
}

func DigitalHumanAvatarOptionsForConfig(defaultAvatarID string, allowSwitch bool) []DigitalHumanAvatarOption {
	if allowSwitch {
		return DigitalHumanAvatarOptions()
	}
	option := DigitalHumanAvatarByID(NormalizeDigitalHumanAvatarID(defaultAvatarID))
	if option == nil {
		return DigitalHumanAvatarOptions()[:1]
	}
	return []DigitalHumanAvatarOption{*option}
}

func IsValidDigitalHumanAvatarID(id string) bool {
	return DigitalHumanAvatarByID(id) != nil
}

func NormalizeDigitalHumanAvatarID(id string) string {
	if IsValidDigitalHumanAvatarID(id) {
		return id
	}
	return DefaultDigitalHumanAvatarID
}

func DigitalHumanAvatarByID(id string) *DigitalHumanAvatarOption {
	for _, option := range digitalHumanAvatarOptions {
		if option.ID == id {
			item := option
			return &item
		}
	}
	return nil
}

func ValidateDigitalHumanAvatarID(id string) error {
	if id == "" || IsValidDigitalHumanAvatarID(id) {
		return nil
	}
	return fmt.Errorf("unknown digital human avatar: %s", id)
}
