package repository

import (
	"github.com/scenic-guide/internal/model"
	"gorm.io/gorm"
)

type SystemSettingRepository struct {
	db *gorm.DB
}

func NewSystemSettingRepository(db *gorm.DB) *SystemSettingRepository {
	return &SystemSettingRepository{db: db}
}

// Get 获取设置值
func (r *SystemSettingRepository) Get(key string) (string, error) {
	var setting model.SystemSetting
	err := r.db.Where("key = ?", key).First(&setting).Error
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

// Set 设置值（存在则更新，不存在则创建）
func (r *SystemSettingRepository) Set(key, value string) error {
	var setting model.SystemSetting
	err := r.db.Where("key = ?", key).First(&setting).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(&model.SystemSetting{Key: key, Value: value}).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(&setting).Update("value", value).Error
}

// GetAll 获取所有设置
func (r *SystemSettingRepository) GetAll() (map[string]string, error) {
	var settings []model.SystemSetting
	err := r.db.Find(&settings).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}

// DigitalHumanConfigRepository

type DigitalHumanConfigRepository struct {
	db *gorm.DB
}

func NewDigitalHumanConfigRepository(db *gorm.DB) *DigitalHumanConfigRepository {
	return &DigitalHumanConfigRepository{db: db}
}

// Get 获取数字人配置（只有一条记录）
func (r *DigitalHumanConfigRepository) Get() (*model.DigitalHumanConfig, error) {
	var config model.DigitalHumanConfig
	err := r.db.First(&config).Error
	if err == gorm.ErrRecordNotFound {
		// 返回默认配置
		config = model.DigitalHumanConfig{
			Name:           "小灵",
			Appearance:     "亲和型国风讲解员",
			Costume:        "古典汉服",
			Style:          "古典汉服",
			Color:          "#D4AF37",
			CultureTheme:   "灵山佛教文化与江南山水意境",
			VoiceType:      "温柔女声",
			VoiceTone:      "温暖、端庄、亲切",
			Speed:          0.8,
			Volume:         80,
			Greeting:       "欢迎来到灵山胜境，我是您的数字导览员小灵。",
			DefaultEmotion: "joy",
			EmotionLevel:   3,
		}
		if createErr := r.db.Create(&config).Error; createErr != nil {
			return nil, createErr
		}
		return &config, nil
	}
	return &config, err
}

// Update 更新数字人配置
func (r *DigitalHumanConfigRepository) Update(config *model.DigitalHumanConfig) error {
	return r.db.Save(config).Error
}
