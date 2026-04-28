package service

import (
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
)

type GuideContentService interface {
	CreateContent(content *model.GuideContent) error
	GetContentByID(id uint) (*model.GuideContent, error)
	GetContentsBySpotID(spotID uint) ([]model.GuideContent, error)
	GetContentsBySpotIDAndType(spotID uint, contentType string) ([]model.GuideContent, error)
	UpdateContent(content *model.GuideContent) error
	DeleteContent(id uint) error
}

type guideContentService struct {
	repo repository.GuideContentRepository
}

func NewGuideContentService(repo repository.GuideContentRepository) GuideContentService {
	return &guideContentService{repo: repo}
}

func (s *guideContentService) CreateContent(content *model.GuideContent) error {
	return s.repo.Create(content)
}

func (s *guideContentService) GetContentByID(id uint) (*model.GuideContent, error) {
	return s.repo.FindByID(id)
}

func (s *guideContentService) GetContentsBySpotID(spotID uint) ([]model.GuideContent, error) {
	return s.repo.FindBySpotID(spotID)
}

func (s *guideContentService) GetContentsBySpotIDAndType(spotID uint, contentType string) ([]model.GuideContent, error) {
	return s.repo.FindBySpotIDAndType(spotID, contentType)
}

func (s *guideContentService) UpdateContent(content *model.GuideContent) error {
	return s.repo.Update(content)
}

func (s *guideContentService) DeleteContent(id uint) error {
	return s.repo.Delete(id)
}
