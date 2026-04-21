package middleware

import (
	"bytes"

	"github.com/gin-gonic/gin"
)

func HeadMethodHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "HEAD" {
			writer := &responseWriter{
				ResponseWriter: c.Writer,
				body:           &bytes.Buffer{},
			}
			c.Writer = writer

			c.Request.Method = "GET"

			c.Next()

			c.Request.Method = "HEAD"

			return
		}

		c.Next()
	}
}

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return len(data), nil
}

func (w *responseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
