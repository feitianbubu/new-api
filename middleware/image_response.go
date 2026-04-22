package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

func ImageResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.ImageRequest
		reqParsed := false
		if strings.HasPrefix(c.ContentType(), gin.MIMEJSON) {
			if body, err := io.ReadAll(c.Request.Body); err == nil {
				_ = common.Unmarshal(body, &req)
				reqParsed = true
				c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			}
		}

		origWriter := c.Writer
		bufWriter := &imageBufferWriter{ResponseWriter: origWriter}
		c.Writer = bufWriter

		c.Next()

		status := bufWriter.status
		if status == 0 {
			status = http.StatusOK
		}

		body := bufWriter.body.Bytes()

		var resp dto.ImageResponse
		if status >= http.StatusBadRequest || len(body) == 0 || common.Unmarshal(body, &resp) != nil {
			writeThrough(origWriter, bufWriter.Header(), status, body)
			return
		}

		info := &relaycommon.RelayInfo{Request: &req}
		if !reqParsed || req.ResponseFormat == "" {
			info.Request = &dto.ImageRequest{ResponseFormat: "url"}
		}

		channel.TryConvertBase64ImagesToURLs(c, &resp)

		out, err := json.Marshal(resp)
		if err != nil {
			writeThrough(origWriter, bufWriter.Header(), status, body)
			return
		}

		h := bufWriter.Header()
		h.Set("Content-Type", "application/json")
		writeThrough(origWriter, h, status, out)
	}
}

type imageBufferWriter struct {
	gin.ResponseWriter
	body   bytes.Buffer
	status int
}

func (w *imageBufferWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
}

func (w *imageBufferWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(data)
}

func (w *imageBufferWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// Flush is intentionally a no-op to avoid premature flushes when
// upstream handlers call http.Flusher on the buffered writer.
func (w *imageBufferWriter) Flush() {}

func writeThrough(dst gin.ResponseWriter, hdr http.Header, status int, body []byte) {
	for k, v := range hdr {
		dst.Header()[k] = v
	}
	dst.Header().Del("Content-Length")
	dst.WriteHeader(status)
	if len(body) > 0 {
		_, _ = dst.Write(body)
	}
}
