package common

import (
	"bytes"
	"io"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

var WhiteLogPaths []string

func init() {
	paths := GetEnvOrDefaultString("WHITE_LOG_PATHS", "/api,/providers,/v1,/pg,/suno")
	WhiteLogPaths = strings.Split(paths, ",")
}

const KeyResponseWriter = "key_response_Writer"

func ShouldLogBody(c *gin.Context) bool {
	if !ResponseLogEnabled {
		return false
	}

	return slices.ContainsFunc(WhiteLogPaths, func(p string) bool {
		return strings.HasPrefix(c.Request.URL.Path, p)
	})
}

type BodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w BodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}
func (w BodyLogWriter) String() string {
	return w.body.String()
}
func (w BodyLogWriter) Bytes() []byte {
	return w.body.Bytes()
}
func WrapWriter(c *gin.Context) *gin.Context {
	if !ShouldLogBody(c) {
		return c
	}
	storage, _ := GetBodyStorage(c)
	if storage != nil {
		if _, err := storage.Seek(0, io.SeekStart); err == nil {
			c.Request.Body = io.NopCloser(ReaderOnly(storage))
		}
	}
	w := BodyLogWriter{
		ResponseWriter: c.Writer,
		body:           bytes.NewBufferString(""),
	}
	c.Writer = w
	c.Set(KeyResponseWriter, w)
	return c
}
