package pkg

import (
	"log/slog"
	"runtime/debug"
)

// SafeGo 在新的 goroutine 中异步执行 fn,并捕获 panic。
//
// 用途:替代裸 `go func(){...}()`。未 recover 的 goroutine panic 会直接终止整个进程,
// 这对长运行的服务是灾难性的。所有后台异步任务(会话持久化、统计记录等)都应通过
// SafeGo 启动,确保单次 panic 只产生错误日志而不会拖垮服务。
//
// name 仅用于日志标识,帮助定位 panic 来源。
func SafeGo(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("异步任务 panic 已恢复",
					"task", name,
					"panic", r,
					"stack", string(debug.Stack()),
				)
			}
		}()
		fn()
	}()
}
