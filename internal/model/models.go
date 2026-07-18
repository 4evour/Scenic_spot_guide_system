package model

import (
	"time"

	"gorm.io/gorm"
)

type ScenicSpot struct {
	ID                      uint      `gorm:"primaryKey" json:"id"`
	Name                    string    `gorm:"size:255;not null" json:"name"`
	Description             string    `gorm:"text" json:"description"`
	Location                string    `gorm:"size:500" json:"location"`
	Category                string    `gorm:"size:100;index:idx_scenic_spots_category_updated" json:"category"`
	Rating                  float64   `gorm:"default:0" json:"rating"`
	Price                   float64   `gorm:"default:0" json:"price"`
	ImageURL                string    `gorm:"size:500" json:"image_url"`
	Latitude                float64   `gorm:"column:latitude;default:0" json:"latitude"`
	Longitude               float64   `gorm:"column:longitude;default:0" json:"longitude"`
	SortOrder               int       `gorm:"column:sort_order;default:0" json:"sort_order"`
	QRCode                  string    `gorm:"size:100;uniqueIndex:idx_scenic_spots_qr_code,where:qr_code != ''" json:"qr_code"`
	QRIntroText             string    `gorm:"text" json:"qr_intro_text"`
	QREnabled               bool      `gorm:"default:false" json:"qr_enabled"`
	GeofenceEnabled         bool      `gorm:"default:false" json:"geofence_enabled"`
	GeofenceRadiusM         int       `gorm:"default:100" json:"geofence_radius_m"`
	GeofenceIntroText       string    `gorm:"text" json:"geofence_intro_text"`
	GeofenceCooldownMinutes int       `gorm:"default:1440" json:"geofence_cooldown_minutes"`
	CreatedAt               time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time `gorm:"autoUpdateTime;index:idx_scenic_spots_category_updated" json:"updated_at"`
}

type GuideContent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SpotID    uint      `gorm:"not null" json:"spot_id"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Content   string    `gorm:"text;not null" json:"content"`
	Type      string    `gorm:"size:50" json:"content_type"`
	AudioURL  string    `gorm:"size:500" json:"audio_url"`
	Duration  int       `gorm:"default:0" json:"duration"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type TourRoute struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	Description string    `gorm:"text" json:"description"`
	Spots       string    `gorm:"text" json:"spots"`
	Duration    int       `gorm:"default:0" json:"duration"`
	Difficulty  string    `gorm:"size:50" json:"difficulty"`
	Rating      float64   `gorm:"default:0" json:"rating"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type VisitorSpotRating struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SessionID      string    `gorm:"size:100;not null;uniqueIndex:idx_visitor_spot_rating_session_spot;index" json:"session_id"`
	SpotID         uint      `gorm:"not null;uniqueIndex:idx_visitor_spot_rating_session_spot;index" json:"spot_id"`
	OverallRating  int       `gorm:"default:0" json:"overall_rating"`
	CultureRating  int       `gorm:"default:0" json:"culture_rating"`
	PhotoRating    int       `gorm:"default:0" json:"photo_rating"`
	FacilityRating int       `gorm:"default:0" json:"facility_rating"`
	Comment        string    `gorm:"text" json:"comment"`
	Tags           string    `gorm:"type:text" json:"tags"`
	Sentiment      string    `gorm:"size:30;index" json:"sentiment"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime;index" json:"updated_at"`
}

type RouteRecommendationLog struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	SessionID     string    `gorm:"size:100;index" json:"session_id"`
	ProfileType   string    `gorm:"size:50;index" json:"profile_type"`
	RouteName     string    `gorm:"size:255;index" json:"route_name"`
	SpotIDs       string    `gorm:"type:text" json:"spot_ids"`
	InterestTags  string    `gorm:"type:text" json:"interest_tags"`
	TotalDuration int       `gorm:"default:0" json:"total_duration"`
	ScoreSummary  string    `gorm:"text" json:"score_summary"`
	CreatedAt     time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

type VisitorQuery struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Query      string    `gorm:"text;not null" json:"query"`
	Response   string    `gorm:"text" json:"response"`
	SpotID     uint      `gorm:"default:0" json:"spot_id"`
	IsAnswered bool      `gorm:"default:false" json:"is_answered"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type User struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Username          string    `gorm:"size:100;uniqueIndex;not null" json:"username"`
	Password          string    `gorm:"size:255;not null" json:"password,omitempty"`
	Email             string    `gorm:"size:255" json:"email"`
	Role              string    `gorm:"size:50;default:'visitor'" json:"role"` // admin | visitor | guest
	TokenVersion      uint      `gorm:"default:0;not null" json:"-"`
	GuestToken        string    `gorm:"size:100;uniqueIndex:idx_guest_token,where:guest_token != ''" json:"-"` // 游客设备绑定标识
	DisplayName       string    `gorm:"size:100" json:"display_name"`                                          // 游客显示名
	PreferredAvatarID string    `gorm:"size:50;default:'mao_pro'" json:"preferred_avatar_id"`                  // 数字人偏好
	CreatedAt         time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type VisitRecord struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null"`
	SpotID    uint      `gorm:"not null"`
	VisitTime time.Time `gorm:"autoCreateTime"`
	Duration  int       `gorm:"default:0"`
}

type SystemLog struct {
	ID        uint      `gorm:"primaryKey"`
	Level     string    `gorm:"size:50"`
	Message   string    `gorm:"text"`
	Source    string    `gorm:"size:255"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// InteractionLog 交互日志 - 记录每次游客与AI的对话
type InteractionLog struct {
	ID             uint      `gorm:"primaryKey"`
	UserID         uint      `gorm:"default:0"`
	SessionID      string    `gorm:"size:100;index:idx_interaction_logs_session"`
	Query          string    `gorm:"text;not null"`
	Response       string    `gorm:"text"`
	Emotion        string    `gorm:"size:50"`
	ResponseTimeMs int64     `gorm:"default:0"` // 响应时间(毫秒)
	SpotID         uint      `gorm:"default:0"`
	Category       string    `gorm:"size:100"`                                          // 问题分类
	Source         string    `gorm:"size:50;index:idx_interaction_logs_source_created"` // 来源: web/voice/digital_human
	CreatedAt      time.Time `gorm:"autoCreateTime;index:idx_interaction_logs_source_created;index:idx_interaction_logs_created"`
}

// ChatSession 聊天会话 - 持久化用户对话
type ChatSession struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index;not null" json:"user_id"`
	SessionID    string    `gorm:"size:100;uniqueIndex;not null" json:"session_id"`
	Title        string    `gorm:"size:255" json:"title"`
	Source       string    `gorm:"size:50;default:'web'" json:"source"` // web | digital_human | api
	MessageCount int       `gorm:"default:0" json:"message_count"`
	LastActiveAt time.Time `gorm:"index" json:"last_active_at"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// ChatMessage 聊天消息 - 持久化每轮对话内容
type ChatMessage struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ChatSessionID  uint      `gorm:"index;not null" json:"chat_session_id"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	Role           string    `gorm:"size:20;not null" json:"role"` // user | assistant | system
	Content        string    `gorm:"text;not null" json:"content"`
	Emotion        string    `gorm:"size:50" json:"emotion"`
	Metadata       string    `gorm:"type:text" json:"metadata,omitempty"`
	ResponseTimeMs int64     `gorm:"default:0" json:"response_time_ms"`
	CreatedAt      time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

type UserFeedback struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	SessionID string    `gorm:"size:100;index" json:"session_id"`
	MessageID uint      `gorm:"index" json:"message_id"`
	TraceID   string    `gorm:"size:100;index" json:"trace_id"`
	Query     string    `gorm:"text" json:"query"`
	Response  string    `gorm:"text" json:"response"`
	Helpful   bool      `gorm:"default:false" json:"helpful"`
	Rating    int       `gorm:"default:0" json:"rating"`
	Reason    string    `gorm:"size:255" json:"reason"`
	Comment   string    `gorm:"text" json:"comment"`
	Source    string    `gorm:"size:50;index" json:"source"`
	SpotID    uint      `gorm:"index" json:"spot_id"`
	CreatedAt time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

type VisitorInsightAnalysis struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            uint      `gorm:"index" json:"user_id"`
	SessionID         string    `gorm:"size:100;index" json:"session_id"`
	Summary           string    `gorm:"text" json:"summary"`
	SatisfactionScore int       `gorm:"default:0" json:"satisfaction_score"`
	NegativeReasons   string    `gorm:"type:text" json:"negative_reasons"`
	AttentionPoints   string    `gorm:"type:text" json:"attention_points"`
	RawResult         string    `gorm:"type:text" json:"raw_result"`
	Status            string    `gorm:"size:30;default:'completed';index" json:"status"`
	CreatedAt         time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type KnowledgeCandidate struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	AnalysisID        uint       `gorm:"index" json:"analysis_id"`
	SessionID         string     `gorm:"size:100;index" json:"session_id"`
	Title             string     `gorm:"size:255;not null" json:"title"`
	Content           string     `gorm:"text;not null" json:"content"`
	Source            string     `gorm:"size:255;default:'chat-insight'" json:"source"`
	KnowledgeCategory string     `gorm:"size:100;index" json:"knowledge_category"`
	SpotID            uint       `gorm:"index" json:"spot_id"`
	SpotCategory      string     `gorm:"size:100;index" json:"spot_category"`
	Status            string     `gorm:"size:30;default:'pending';index" json:"status"`
	RejectReason      string     `gorm:"text" json:"reject_reason"`
	CreatedAt         time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	ApprovedAt        *time.Time `json:"approved_at,omitempty"`
}

// SystemSetting 系统设置 - 键值对存储
type SystemSetting struct {
	ID        uint      `gorm:"primaryKey"`
	Key       string    `gorm:"size:255;unique;not null"`
	Value     string    `gorm:"text"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// DigitalHumanConfig 数字人配置
type DigitalHumanConfig struct {
	ID                uint      `gorm:"primaryKey"`
	Name              string    `gorm:"size:100;default:'小灵'"`
	Appearance        string    `gorm:"size:255;default:'Live2D 官方示例形象（临时）'"`
	Costume           string    `gorm:"size:255;default:'现代幻想服装'"`
	Style             string    `gorm:"size:100;default:'Live2D 示例模型'"`
	Color             string    `gorm:"size:20;default:'#D4AF37'"`
	CultureTheme      string    `gorm:"size:255;default:'灵山佛教文化与江南山水意境'"`
	VoiceType         string    `gorm:"size:100;default:'温柔女声'"`
	VoiceTone         string    `gorm:"size:100;default:'温暖、端庄、亲切'"`
	VoiceID           string    `gorm:"size:100;default:'female_xiaoxiao'"`
	TTSRate           string    `gorm:"size:20;default:''"`
	Speed             float64   `gorm:"default:0.8"`
	Volume            int       `gorm:"default:80"`
	Greeting          string    `gorm:"size:500;default:'欢迎来到灵山胜境，我是您的数字导览员小灵。'"`
	DefaultEmotion    string    `gorm:"size:50;default:'joy'"`
	EmotionLevel      int       `gorm:"default:3"`
	DefaultAvatarID   string    `gorm:"size:50;default:'mao_pro'"`
	AllowAvatarSwitch bool      `gorm:"default:true"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime"`
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&ScenicSpot{},
		&GuideContent{},
		&TourRoute{},
		&VisitorSpotRating{},
		&RouteRecommendationLog{},
		&VisitorQuery{},
		&User{},
		&VisitRecord{},
		&SystemLog{},
		&KnowledgeChunk{},
		&InteractionLog{},
		&SystemSetting{},
		&DigitalHumanConfig{},
		&ChatSession{},
		&ChatMessage{},
		&UserFeedback{},
		&VisitorInsightAnalysis{},
		&KnowledgeCandidate{},
	)
}
