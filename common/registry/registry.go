// Package registry 提供主线挂载点的通用注册表，
// 让额外的 router 通过 init() 自注册接入主流程。
package registry

import (
	"sync"

	"github.com/gin-gonic/gin"
)

type (
	RouterFunc func(*gin.Engine)
	InitFunc   func()
)

var (
	mu          sync.Mutex
	routerFuncs []RouterFunc
	initFuncs   []InitFunc
)

func RegisterRouter(f RouterFunc) {
	mu.Lock()
	routerFuncs = append(routerFuncs, f)
	mu.Unlock()
}

func RegisterInit(f InitFunc) {
	mu.Lock()
	initFuncs = append(initFuncs, f)
	mu.Unlock()
}

// ApplyRouters 复制 + 释放锁后再执行，避免 hook 内 Register* 触发自死锁。
func ApplyRouters(r *gin.Engine) {
	mu.Lock()
	fs := append([]RouterFunc(nil), routerFuncs...)
	mu.Unlock()
	for _, f := range fs {
		f(r)
	}
}

func RunInits() {
	mu.Lock()
	fs := append([]InitFunc(nil), initFuncs...)
	mu.Unlock()
	for _, f := range fs {
		f()
	}
}
