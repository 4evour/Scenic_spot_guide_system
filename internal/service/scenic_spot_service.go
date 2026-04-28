package service

import (
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
)

type ScenicSpotService interface {
	CreateSpot(spot *model.ScenicSpot) error
	GetSpotByID(id uint) (*model.ScenicSpot, error)
	GetAllSpots() ([]model.ScenicSpot, error)
	GetSpotsByCategory(category string) ([]model.ScenicSpot, error)
	UpdateSpot(spot *model.ScenicSpot) error
	DeleteSpot(id uint) error
}

type scenicSpotService struct {
	repo repository.ScenicSpotRepository
}

func NewScenicSpotService(repo repository.ScenicSpotRepository) ScenicSpotService {
	return &scenicSpotService{repo: repo}
}

func (s *scenicSpotService) CreateSpot(spot *model.ScenicSpot) error {
	return s.repo.Create(spot)
}

func (s *scenicSpotService) GetSpotByID(id uint) (*model.ScenicSpot, error) {
	return s.repo.FindByID(id)
}

func (s *scenicSpotService) GetAllSpots() ([]model.ScenicSpot, error) {
	return s.repo.FindAll()
}

func (s *scenicSpotService) GetSpotsByCategory(category string) ([]model.ScenicSpot, error) {
	return s.repo.FindByCategory(category)
}

func (s *scenicSpotService) UpdateSpot(spot *model.ScenicSpot) error {
	return s.repo.Update(spot)
}

func (s *scenicSpotService) DeleteSpot(id uint) error {
	return s.repo.Delete(id)
}
