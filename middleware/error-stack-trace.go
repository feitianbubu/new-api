package middleware

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

var (
	projectRoot     string
	projectRootOnce sync.Once
	moduleName      string
	moduleNameOnce  sync.Once
)

type errorTrackingWriter struct {
	gin.ResponseWriter
	ctx           *gin.Context
	statusWritten bool
}

func (w *errorTrackingWriter) WriteHeader(code int) {
	if !w.statusWritten {
		w.statusWritten = true

		if code >= 400 {
			stack := getBusinessStack()
			requestID := ""
			if val, exists := w.ctx.Get(common.RequestIdKey); exists {
				requestID = val.(string)
			}

			common.SysLog(fmt.Sprintf("%s | [STAICK] %d\n%s",
				requestID,
				code,
				stack,
			))
		}
	}

	w.ResponseWriter.WriteHeader(code)
}

func (w *errorTrackingWriter) Write(data []byte) (int, error) {
	if !w.statusWritten {
		w.statusWritten = true
	}
	return w.ResponseWriter.Write(data)
}

func getBusinessStack() string {
	var lines []string
	var foundBusiness bool

	for i := 1; i <= 20; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		funcName := runtime.FuncForPC(pc).Name()

		if shouldSkipFrame(funcName, file) {
			continue
		}
		foundBusiness = true
		displayFile := getRelativePath(file)
		displayFunc := trimModuleName(funcName)
		lines = append(lines, fmt.Sprintf("    %s:%d %s", displayFile, line, displayFunc))
		if len(lines) >= 8 {
			break
		}
	}

	if !foundBusiness || len(lines) == 0 {
		return "    (no business code stack trace available)"
	}

	return strings.Join(lines, "\n")
}

func shouldSkipFrame(funcName, file string) bool {
	modName := getModuleName()
	if modName != "" && !strings.Contains(funcName, modName) {
		return true
	}

	if strings.Contains(file, "middleware/error-stack-trace.go") {
		return true
	}

	return false
}

func getModuleName() string {
	moduleNameOnce.Do(func() {
		pc, _, _, ok := runtime.Caller(0)
		if ok {
			fullName := runtime.FuncForPC(pc).Name()
			if idx := strings.LastIndex(fullName, "/middleware."); idx != -1 {
				moduleName = fullName[:idx]
			}
		}
	})
	return moduleName
}

func getProjectRoot() string {
	projectRootOnce.Do(func() {
		_, file, _, ok := runtime.Caller(0)
		if ok {
			projectRoot = filepath.Dir(filepath.Dir(file))
		}
	})
	return projectRoot
}

func getRelativePath(absPath string) string {
	root := getProjectRoot()
	if root != "" && strings.HasPrefix(absPath, root) {
		relPath := strings.TrimPrefix(absPath, root)
		relPath = strings.TrimPrefix(relPath, "/")
		return relPath
	}
	return absPath
}

func trimModuleName(fullFuncName string) string {
	modName := getModuleName()
	if modName != "" {
		prefix := modName + "/"
		return strings.TrimPrefix(fullFuncName, prefix)
	}
	return fullFuncName
}

func ErrorStackTrace() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer = &errorTrackingWriter{
			ResponseWriter: c.Writer,
			ctx:            c,
			statusWritten:  false,
		}

		c.Next()
	}
}
