package service

import (
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
)

type TourRouteService interface {
	CreateRoute(route *model.TourRoute) error
	GetRouteByID(id uint) (*model.TourRoute, error)
	GetAllRoutes() ([]model.TourRoute, error)
	GetRoutesByDifficulty(difficulty string) ([]model.TourRoute, error)
	UpdateRoute(route *model.TourRoute) error
	DeleteRoute(id uint) error
}

type tourRouteService struct {
	repo repository.TourRouteRepository
}

func NewTourRouteService(repo repository.TourRouteRepository) TourRouteService {
	return &tourRouteService{repo: repo}
}

func (s *tourRouteService) CreateRoute(route *model.TourRoute) error {
	return s.repo.Create(route)
}

func (s *tourRouteService) GetRouteByID(id uint) (*model.TourRoute, error) {
	return s.repo.FindByID(id)
}

func (s *tourRouteService) GetAllRoutes() ([]model.TourRoute, error) {
	return s.repo.FindAll()
}

func (s *tourRouteService) GetRoutesByDifficulty(difficulty string) ([]model.TourRoute, error) {
	return s.repo.FindByDifficulty(difficulty)
}

func (s *tourRouteService) UpdateRoute(route *model.TourRoute) error {
	return s.repo.Update(route)
}

func (s *tourRouteService) DeleteRoute(id uint) error {
	return s.repo.Delete(id)
}
