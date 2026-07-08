package service

import (
	"errors"

	"fmt"

	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService interface {
	Register(user *model.User) error
	CreateUser(user *model.User) error
	Login(username, password string) (*model.User, error)
	ChangePassword(id uint, currentPassword, newPassword string) error
	GetUserByID(id uint) (*model.User, error)
	GetUserByUsername(username string) (*model.User, error)
	UpdateUser(user *model.User) error
	UpdateAdminUser(id uint, username, email, role, password *string) (*model.User, error)
	UpdateProfile(id uint, username, email string) error
	GetAvatarPreference(id uint) (string, error)
	UpdateAvatarPreference(id uint, avatarID string) error
	DeleteUser(id uint) error
	GetAllUsers() ([]model.User, error)
	GetAllUsersPaginated(page, pageSize int) ([]model.User, int64, error)
	GetUsersByRole(role string) ([]model.User, error)
}

type userService struct {
	repo repository.UserRepository
}

var ErrInvalidCurrentPassword = errors.New("当前密码错误")

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) Register(user *model.User) error {
	existingUser, err := s.repo.FindByUsername(user.Username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询用户名失败: %w", err)
	}
	if existingUser != nil && existingUser.ID != 0 {
		return errors.New("用户名已存在")
	}

	if err := validatePassword(user.Password); err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)

	return s.repo.Create(user)
}

func (s *userService) CreateUser(user *model.User) error {
	if user.Role == "" {
		user.Role = "visitor"
	}
	return s.Register(user)
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("密码长度不能少于8个字符")
	}
	if len(password) > 128 {
		return errors.New("密码长度不能超过128个字符")
	}
	var hasUpper, hasLower, hasDigit bool
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return errors.New("密码必须包含大写字母、小写字母和数字")
	}
	return nil
}

func (s *userService) Login(username, password string) (*model.User, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	return user, nil
}

func (s *userService) ChangePassword(id uint, currentPassword, newPassword string) error {
	user, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return ErrInvalidCurrentPassword
	}
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdateFields(id, map[string]interface{}{"password": string(hashedPassword)})
}

func (s *userService) GetUserByID(id uint) (*model.User, error) {
	return s.repo.FindByID(id)
}

func (s *userService) GetUserByUsername(username string) (*model.User, error) {
	return s.repo.FindByUsername(username)
}

func (s *userService) UpdateUser(user *model.User) error {
	return s.repo.Update(user)
}

func (s *userService) UpdateAdminUser(id uint, username, email, role, password *string) (*model.User, error) {
	user, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	fields := map[string]interface{}{}
	if username != nil {
		fields["username"] = *username
		user.Username = *username
	}
	if email != nil {
		fields["email"] = *email
		user.Email = *email
	}
	if role != nil {
		fields["role"] = *role
		user.Role = *role
	}
	if password != nil && *password != "" {
		if err := validatePassword(*password); err != nil {
			return nil, err
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		fields["password"] = string(hashedPassword)
	}
	if len(fields) == 0 {
		return user, nil
	}
	if err := s.repo.UpdateFields(id, fields); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *userService) UpdateProfile(id uint, username, email string) error {
	return s.repo.UpdateFields(id, map[string]interface{}{
		"username": username,
		"email":    email,
	})
}

func (s *userService) GetAvatarPreference(id uint) (string, error) {
	user, err := s.repo.FindByID(id)
	if err != nil {
		return "", err
	}
	return NormalizeDigitalHumanAvatarID(user.PreferredAvatarID), nil
}

func (s *userService) UpdateAvatarPreference(id uint, avatarID string) error {
	if err := ValidateDigitalHumanAvatarID(avatarID); err != nil {
		return err
	}
	return s.repo.UpdateFields(id, map[string]interface{}{
		"preferred_avatar_id": NormalizeDigitalHumanAvatarID(avatarID),
	})
}

func (s *userService) DeleteUser(id uint) error {
	return s.repo.Delete(id)
}

func (s *userService) GetAllUsers() ([]model.User, error) {
	return s.repo.FindAll()
}

func (s *userService) GetAllUsersPaginated(page, pageSize int) ([]model.User, int64, error) {
	return s.repo.FindAllPaginated(page, pageSize)
}

func (s *userService) GetUsersByRole(role string) ([]model.User, error) {
	return s.repo.FindByRole(role)
}
