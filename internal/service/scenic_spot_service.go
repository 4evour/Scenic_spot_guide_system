package service

import (
	"math"
	"sort"

	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
)

// NearbySpot 带距离信息的景点
type NearbySpot struct {
	model.ScenicSpot
	DistanceMeters float64 `json:"distance_meters"`
}

type ScenicSpotService interface {
	CreateSpot(spot *model.ScenicSpot) error
	GetSpotByID(id uint) (*model.ScenicSpot, error)
	GetAllSpots() ([]model.ScenicSpot, error)
	GetSpotsByCategory(category string) ([]model.ScenicSpot, error)
	GetNearbySpots(lat, lng float64, radiusMeters float64) ([]NearbySpot, error)
	GetSpotByQRCode(code string) (*model.ScenicSpot, error)
	GetAllSpotsWithQR() ([]model.ScenicSpot, error)
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

func (s *scenicSpotService) GetSpotByQRCode(code string) (*model.ScenicSpot, error) {
	return s.repo.FindByQRCode(code)
}

func (s *scenicSpotService) GetAllSpotsWithQR() ([]model.ScenicSpot, error) {
	return s.repo.FindAllWithQR()
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

func (s *scenicSpotService) GetNearbySpots(lat, lng float64, radiusMeters float64) ([]NearbySpot, error) {
	allSpots, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}

	var result []NearbySpot
	for _, spot := range allSpots {
		dist := haversineDistance(lat, lng, spot.Latitude, spot.Longitude)
		if dist <= radiusMeters {
			result = append(result, NearbySpot{
				ScenicSpot:     spot,
				DistanceMeters: math.Round(dist*10) / 10, // 保留一位小数
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].DistanceMeters < result[j].DistanceMeters
	})

	return result, nil
}

// haversineDistance 使用 Haversine 公式计算两点球面距离（米）
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000.0 // 地球半径（米）

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLng := (lng2 - lng1) * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
