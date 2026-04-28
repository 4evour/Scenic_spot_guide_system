package service

import (
	"errors"

	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	Register(user *model.User) error
	Login(username, password string) (*model.User, error)
	GetUserByID(id uint) (*model.User, error)
	GetUserByUsername(username string) (*model.User, error)
	UpdateUser(user *model.User) error
	DeleteUser(id uint) error
	GetAllUsers() ([]model.User, error)
	GetUsersByRole(role string) ([]model.User, error)
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) Register(user *model.User) error {
	existingUser, _ := s.repo.FindByUsername(user.Username)
	if existingUser.ID != 0 {
		return errors.New("用户名已存在")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)

	return s.repo.Create(user)
}

func (s *userService) Login(username, password string) (*model.User, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("密码错误")
	}

	return user, nil
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

func (s *userService) DeleteUser(id uint) error {
	return s.repo.Delete(id)
}

func (s *userService) GetAllUsers() ([]model.User, error) {
	return s.repo.FindAll()
}

func (s *userService) GetUsersByRole(role string) ([]model.User, error) {
	return s.repo.FindByRole(role)
}
