package pkg

import "github.com/scenic-guide/internal/service"

// statsService 是包级别的统计服务实例，通过 SetStatsService 初始化
var statsService *service.StatsService

// SetStatsService 初始化全局统计服务实例
func SetStatsService(s *service.StatsService) {
	statsService = s
}

// GetStatsService 获取全局统计服务实例（未初始化时返回 nil）
func GetStatsService() *service.StatsService {
	return statsService
}
