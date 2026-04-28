package service

import (
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
)

type VisitorQueryService interface {
	CreateQuery(query *model.VisitorQuery) error
	GetQueryByID(id uint) (*model.VisitorQuery, error)
	GetAllQueries() ([]model.VisitorQuery, error)
	GetUnansweredQueries() ([]model.VisitorQuery, error)
	UpdateQuery(query *model.VisitorQuery) error
	DeleteQuery(id uint) error
}

type visitorQueryService struct {
	repo repository.VisitorQueryRepository
}

func NewVisitorQueryService(repo repository.VisitorQueryRepository) VisitorQueryService {
	return &visitorQueryService{repo: repo}
}

func (s *visitorQueryService) CreateQuery(query *model.VisitorQuery) error {
	return s.repo.Create(query)
}

func (s *visitorQueryService) GetQueryByID(id uint) (*model.VisitorQuery, error) {
	return s.repo.FindByID(id)
}

func (s *visitorQueryService) GetAllQueries() ([]model.VisitorQuery, error) {
	return s.repo.FindAll()
}

func (s *visitorQueryService) GetUnansweredQueries() ([]model.VisitorQuery, error) {
	return s.repo.FindUnanswered()
}

func (s *visitorQueryService) UpdateQuery(query *model.VisitorQuery) error {
	return s.repo.Update(query)
}

func (s *visitorQueryService) DeleteQuery(id uint) error {
	return s.repo.Delete(id)
}
