package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"

	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// GuestService 游客账号管理服务
type GuestService struct {
	userRepo         repository.UserRepository
	sessionRepo      repository.ChatSessionRepository
	tokenExpireHours int
}

func NewGuestService(userRepo repository.UserRepository, sessionRepo repository.ChatSessionRepository, tokenExpireHours int) *GuestService {
	if tokenExpireHours <= 0 {
		tokenExpireHours = 24 // 游客 token 默认 24 小时
	}
	return &GuestService{
		userRepo:         userRepo,
		sessionRepo:      sessionRepo,
		tokenExpireHours: tokenExpireHours,
	}
}

// CreateGuestAccount 根据设备指纹创建或恢复游客账号，返回 user 和 JWT token
func (gs *GuestService) CreateGuestAccount(deviceFingerprint string) (*model.User, string, error) {
	if deviceFingerprint == "" {
		return nil, "", errors.New("device fingerprint cannot be empty")
	}

	// 尝试根据指纹查找已有游客
	existing, err := gs.userRepo.FindByGuestToken(deviceFingerprint)
	if err == nil && existing != nil && existing.ID != 0 {
		// 已有游客账号，签发 token
		token, tokenErr := pkg.GenerateToken(existing.ID, existing.Username, existing.Role, existing.TokenVersion, gs.tokenExpireHours)
		if tokenErr != nil {
			return nil, "", fmt.Errorf("generate token failed: %w", tokenErr)
		}
		return existing, token, nil
	}

	// 创建新游客账号
	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		return nil, "", fmt.Errorf("generate random bytes failed: %w", err)
	}
	suffix := hex.EncodeToString(randBytes)[:8]
	username := "guest_" + suffix
	displayName := fmt.Sprintf("游客%s", suffix[:4])

	// 生成随机密码（游客不直接用密码登录）
	randomPwd := make([]byte, 16)
	if _, err := rand.Read(randomPwd); err != nil {
		return nil, "", fmt.Errorf("generate random password failed: %w", err)
	}
	hashedPwd, err := bcrypt.GenerateFromPassword(randomPwd, bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("hash password failed: %w", err)
	}

	user := &model.User{
		Username:    username,
		Password:    string(hashedPwd),
		Role:        "guest",
		GuestToken:  deviceFingerprint,
		DisplayName: displayName,
	}

	if err := gs.userRepo.Create(user); err != nil {
		return nil, "", fmt.Errorf("create guest user failed: %w", err)
	}

	slog.Info("创建游客账号", "user_id", user.ID, "username", username, "display_name", displayName)

	token, err := pkg.GenerateToken(user.ID, user.Username, user.Role, user.TokenVersion, gs.tokenExpireHours)
	if err != nil {
		return nil, "", fmt.Errorf("generate token failed: %w", err)
	}

	return user, token, nil
}

// UpgradeGuest 游客升级为正式账号，迁移会话数据
func (gs *GuestService) UpgradeGuest(userID uint, username, password, email string) (*model.User, error) {
	// 校验用户名
	if len(username) < 3 || len(username) > 32 {
		return nil, errors.New("用户名长度必须在 3-32 之间")
	}
	// 校验密码强度
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	// 检查用户名是否已被使用
	existing, err := gs.userRepo.FindByUsername(username)
	if err == nil && existing != nil && existing.ID != 0 {
		return nil, errors.New("用户名已被使用")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) && err != nil {
		return nil, fmt.Errorf("查询用户名失败: %w", err)
	}

	// 检查邮箱
	if email != "" {
		existingEmail, err := gs.userRepo.FindByEmail(email)
		if err == nil && existingEmail != nil && existingEmail.ID != 0 {
			return nil, errors.New("邮箱已被使用")
		}
	}

	// 哈希密码
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password failed: %w", err)
	}

	fields := map[string]interface{}{
		"username":      username,
		"password":      string(hashedPwd),
		"role":          "visitor",
		"email":         email,
		"display_name":  "",
		"token_version": gorm.Expr("token_version + ?", 1),
	}

	if err := gs.userRepo.UpgradeGuest(userID, fields); err != nil {
		return nil, fmt.Errorf("升级游客失败: %w", err)
	}

	// 获取更新后的用户
	user, err := gs.userRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	slog.Info("游客升级为正式账号", "user_id", userID, "username", username)
	return user, nil
}

// generateRandomHex 生成指定长度的随机十六进制字符串
func generateRandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
