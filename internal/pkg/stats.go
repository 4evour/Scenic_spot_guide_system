package pkg

// statsService 是包级别的统计服务实例，通过 SetStatsService 初始化
var statsService interface{}

// SetStatsService 初始化全局统计服务实例
func SetStatsService(s interface{}) {
	statsService = s
}

// GetStatsService 获取全局统计服务实例（未初始化时返回 nil）
func GetStatsService() interface{} {
	return statsService
}
