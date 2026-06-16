package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/config"
	"github.com/scenic-guide/internal/pkg"
)

type ScenicProfileHandler struct {
	profile *config.ScenicProfile
}

func NewScenicProfileHandler(profile *config.ScenicProfile) *ScenicProfileHandler {
	return &ScenicProfileHandler{profile: profile}
}

func (h *ScenicProfileHandler) GetProfile(c *gin.Context) {
	if h.profile == nil {
		pkg.InternalError(c, pkg.T(c, "msg_profile_not_loaded"))
		return
	}
	pkg.Success(c, gin.H{
		"id":          h.profile.ID,
		"name":        h.profile.Name,
		"short_name":  h.profile.ShortName,
		"slogan":      h.profile.Slogan,
		"description": h.profile.Description,
		"hotline":     h.profile.Hotline,
		"digital_human": gin.H{
			"name":        h.profile.DigitalHuman.Name,
			"greeting":    h.profile.DigitalHuman.Greeting,
			"personality": h.profile.DigitalHuman.Personality,
			"tts_voice":   h.profile.DigitalHuman.TTSVoice,
			"tts_speed":   h.profile.DigitalHuman.TTSSpeed,
		},
		"quick_asks":      h.getQuickAsks(),
		"routes":          h.GetRoutes(),
		"topic_entities":  h.getTopicEntities(),
	})
}

func (h *ScenicProfileHandler) GetQuickAsks(c *gin.Context) {
	pkg.Success(c, h.getQuickAsks())
}

func (h *ScenicProfileHandler) GetPersonaPrompt(c *gin.Context) {
	if h.profile == nil {
		pkg.InternalError(c, pkg.T(c, "msg_profile_not_loaded"))
		return
	}

	spots := ""
	for i, kw := range h.profile.Keywords.Spots {
		if i > 0 {
			spots += "、"
		}
		spots += kw
	}

	prompt := h.profile.Prompts.SystemRole
	prompt = replacePlaceholder(prompt, "scenic.name", h.profile.Name)
	prompt = replacePlaceholder(prompt, "digital_human.name", h.profile.DigitalHuman.Name)
	prompt = replacePlaceholder(prompt, "digital_human.personality", h.profile.DigitalHuman.Personality)
	prompt = replacePlaceholder(prompt, "spots_list", spots)

	pkg.Success(c, gin.H{
		"persona_prompt": prompt,
		"greeting":       h.profile.DigitalHuman.Greeting,
		"voice":          h.profile.DigitalHuman.TTSVoice,
		"rate":           h.profile.DigitalHuman.TTSSpeed,
	})
}

func (h *ScenicProfileHandler) GetRoutes() []gin.H {
	if h.profile == nil || len(h.profile.Routes) == 0 {
		return nil
	}
	var routes []gin.H
	for _, r := range h.profile.Routes {
		routes = append(routes, gin.H{
			"name":        r.Name,
			"description": r.Description,
			"spots":       r.Spots,
			"duration":    r.Duration,
			"difficulty":  r.Difficulty,
			"rating":      r.Rating,
		})
	}
	return routes
}

func (h *ScenicProfileHandler) getTopicEntities() []string {
	if h.profile == nil {
		return nil
	}
	return h.profile.Keywords.TopicEntities
}

func (h *ScenicProfileHandler) getQuickAsks() []string {
	asks := []string{
		"推荐一条游览路线",
		"带孩子怎么玩？",
		"开放时间是什么？",
		"有什么好吃的？",
	}

	// Dynamically add entity-specific questions from profile topic entities
	if h.profile != nil {
		entities := h.profile.Keywords.TopicEntities
		for i, entity := range entities {
			if i >= 2 {
				break
			}
			asks = append(asks, entity+"有什么特色？")
		}
		if len(asks) < 6 && h.profile.ShortName != "" {
			asks = append([]string{h.profile.ShortName + "有哪些必看景点？"}, asks...)
		}
	}

	if len(asks) > 6 {
		asks = asks[:6]
	}
	return asks
}

func (h *ScenicProfileHandler) Routes(r *gin.RouterGroup) {
	scenic := r.Group("/scenic")
	{
		scenic.GET("/profile", h.GetProfile)
		scenic.GET("/quick-asks", h.GetQuickAsks)
		scenic.GET("/persona", h.GetPersonaPrompt)
	}
}

func replacePlaceholder(s, key, value string) string {
	result := ""
	search := "{" + key + "}"
	for {
		idx := strings.Index(s, search)
		if idx == -1 {
			result += s
			break
		}
		result += s[:idx] + value
		s = s[idx+len(search):]
	}
	return result
}

