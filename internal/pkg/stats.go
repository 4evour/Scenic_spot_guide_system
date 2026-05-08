package pkg

import "github.com/scenic-guide/internal/service"

// StatsService 全局统计服务实例，供各 handler 记录交互日志
var StatsService *service.StatsService
