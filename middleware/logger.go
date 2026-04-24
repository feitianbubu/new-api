package middleware

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

const (
	maxSize = 1024
)

const RouteTagKey = "route_tag"

func RouteTag(tag string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(RouteTagKey, tag)
		c.Next()
	}
}

func SetUpLogger(server *gin.Engine) {
	server.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		if common.ResponseLogEnabled {
			return ""
		}
		var requestID string
		if param.Keys != nil {
			requestID, _ = param.Keys[common.RequestIdKey].(string)
		}
		tag, _ := param.Keys[RouteTagKey].(string)
		if tag == "" {
			tag = "web"
		}
		return fmt.Sprintf("[GIN] %s | %s | %s | %3d | %13v | %15s | %7s %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			tag,
			requestID,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
		)
	}))

	server.Use(func(c *gin.Context) {
		c = common.WrapWriter(c)
		c.Next()
	})

	server.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method

		if c.ContentType() == gin.MIMEMultipartPOSTForm {
			_ = c.Request.ParseMultipartForm(int64(constant.MaxRequestBodyMB) << 20)
		}

		reqBody, err := common.GetRequestBodyBytes(c)
		if err != nil {
			logger.LogError(c, fmt.Sprintf("failed to get request body: %v", err))
			return
		}

		multipartPreview := ""
		if common.ResponseLogEnabled && common.OSSLogEnabled && c.ContentType() == gin.MIMEMultipartPOSTForm {
			var previewErr error
			multipartPreview, previewErr = buildMultipartPreviewJSON(c)
			if previewErr != nil {
				logger.LogWarn(c, fmt.Sprintf("failed to unmarshal multipart form for preview: %v", previewErr))
			}
		}

		if common.ResponseLogEnabled {
			logRequestParams(c)
		}

		var logoutUid int
		var logoutUname string
		if path == "/api/user/logout" {
			sess := sessions.Default(c)
			logoutUid, _ = sess.Get("id").(int)
			logoutUname, _ = sess.Get("username").(string)
		}

		c.Next()

		respBody := ""
		if writer, ok := c.Get(common.KeyResponseWriter); ok {
			if blw, ok := writer.(common.BodyLogWriter); ok {
				respBody = blw.String()
			}
		}
		respBody = maybeDecompressGzip(respBody)

		if common.ShouldLogBody(c) {
			resPreview := lo.Substring(respBody, 0, maxSize)
			if resPreview == "" {
				resPreview = "<empty>"
			}
			logger.LogInfo(c.Request.Context(), fmt.Sprintf("[RES] %s", resPreview))

			if common.OSSLogEnabled && common.GetContextKeyBool(c, constant.ContextKeyEnableOssUpload) {
				// 请求体独立上传：client_gone / 上游未返回数据时 respBody 为空，此时仍需请求侧归档以便排查
				safeReqBody := sanitizeRequestBodyForLog(reqBody, c.ContentType())
				model.EnqueueOSSUploadSingle(c, []byte(safeReqBody), c.ContentType(), "input")
				if channelReqBody := common.GetContextKeyString(c, constant.ContextKeyChannelRequestBody); channelReqBody != "" {
					model.EnqueueOSSUploadSingle(c, []byte(channelReqBody), "", "channel_input")
				}
				if c.ContentType() == gin.MIMEMultipartPOSTForm {
					if multipartPreview != "" {
						model.EnqueueOSSUploadSingle(c, []byte(common.SanitizeStringForLog(multipartPreview)), "application/json", "input_preview")
					}

					mf := c.Request.MultipartForm
					if mf != nil {
						for _, fileHeaders := range mf.File {
							for _, fileHeader := range fileHeaders {
								src, err := fileHeader.Open()
								if err != nil {
									logger.LogWarn(c, fmt.Sprintf("failed to open file %s: %v", fileHeader.Filename, err))
									continue
								}
								fileData, err := io.ReadAll(src)
								src.Close()
								if err != nil {
									logger.LogWarn(c, fmt.Sprintf("failed to read file %s: %v", fileHeader.Filename, err))
									continue
								}
								model.EnqueueOSSUploadSinglePreserveName(c, fileData, fileHeader.Header.Get("Content-Type"), fileHeader.Filename)
							}
						}
					}
				}
				model.EnqueueOSSUploadSingle(c, []byte(respBody), c.Writer.Header().Get("Content-Type"), "output")
			}
		}

		var (
			needLog  bool
			logUid   int
			logType  int
			logCont  string
			logOther map[string]interface{}
		)

		statusOK := c.Writer.Status() == http.StatusOK

		if !statusOK {
			return
		}

		loginRegex := regexp.MustCompile(`^/api/oauth/[^/]+$`)
		isLoginPath := path == "/api/user/login" || loginRegex.MatchString(path)

		switch {
		// Login
		case isLoginPath:
			var lr struct {
				Success bool `json:"success"`
				Data    struct {
					Id       int    `json:"id"`
					Username string `json:"username"`
				} `json:"data"`
			}
			if err := common.UnmarshalJsonStr(respBody, &lr); err == nil && lr.Success {
				needLog = true
				logType = model.LogTypeSystem
				logUid = lr.Data.Id
				logCont = fmt.Sprintf("User %s login", lr.Data.Username)
				logOther = map[string]any{"req": sanitizeRequestBodyForLog(reqBody, c.ContentType()), "res": respBody, "login_path": path}
			}

		case path == "/api/user/logout" && logoutUid != 0:
			needLog = true
			logType = model.LogTypeSystem
			logUid = logoutUid
			logCont = fmt.Sprintf("User %s logout", logoutUname)

		case strings.HasPrefix(path, "/api/channel") && (method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete):
			needLog = true
			logType = model.LogTypeManage
			logUid = c.GetInt("id")
			logCont = fmt.Sprintf("Operation: [%s] %s", method, path)
			logOther = map[string]any{"req": sanitizeRequestBodyForLog(reqBody, c.ContentType()), "res": respBody}
		}

		if needLog {
			model.RecordDetailLog(c, logUid, logType, logCont, logOther)
		}
	})
}

func maybeDecompressGzip(body string) string {
	if reader, err := gzip.NewReader(bytes.NewReader([]byte(body))); err == nil {
		if decompressed, err := io.ReadAll(reader); err == nil {
			body = string(decompressed)
		}
		_ = reader.Close()
	}
	return body
}

func logRequestParams(c *gin.Context) {
	if !common.ShouldLogBody(c) {
		return
	}
	params := make(map[string]interface{})
	if len(c.Request.URL.RawQuery) > 0 {
		queryParams := make(map[string]interface{})
		for k, v := range c.Request.URL.Query() {
			if len(v) == 1 {
				queryParams[k] = v[0]
			} else {
				queryParams[k] = v
			}
		}
		params["query"] = queryParams
	}

	if len(c.Params) > 0 {
		pathParams := make(map[string]string)
		for _, p := range c.Params {
			pathParams[p.Key] = p.Value
		}
		params["path"] = pathParams
	}

	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "application/json") {
		requestBody, err := common.GetRequestBodyBytes(c)
		if err == nil && len(requestBody) > 0 {
			var bodyMap map[string]interface{}
			if err := common.Unmarshal(requestBody, &bodyMap); err == nil {
				params["body"] = bodyMap
			} else {
				bodyStr := string(requestBody)
				if len(bodyStr) > maxSize {
					bodyStr = bodyStr[:maxSize] + "..."
				}
				params["body"] = common.SanitizeStringForLog(bodyStr)
			}
		}
	} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if err := c.Request.ParseForm(); err == nil {
			formParams := make(map[string]interface{})
			for k, v := range c.Request.PostForm {
				if len(v) == 1 {
					formParams[k] = v[0]
				} else {
					formParams[k] = v
				}
			}
			params["form"] = formParams
		}
	} else if strings.Contains(contentType, "multipart/form-data") {
		form, err := common.ParseMultipartFormReusable(c)
		if err == nil {
			params["multipart"] = buildMultipartPreviewData(c, form)
		}
	}

	var paramsStr string
	if len(params) > 0 {
		safeParams := common.SanitizeForLog(params)
		truncatedParams := common.TruncateValue(safeParams)

		if paramsJSON, err := common.Marshal(truncatedParams); err == nil {
			paramsStr = string(paramsJSON)
		} else {
			paramsStr = fmt.Sprintf("%+v", truncatedParams)
		}
	}
	if len(paramsStr) > maxSize*2 {
		paramsStr = paramsStr[:maxSize*2] + "..."
	}

	msg := fmt.Sprintf("[REQ] %s %s %s %s",
		c.ClientIP(),
		c.Request.Method,
		c.Request.URL.Path,
		paramsStr)
	logger.LogInfo(c.Request.Context(), msg)
}

func buildMultipartPreviewJSON(c *gin.Context) (string, error) {
	form, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return "", err
	}
	return common.MapToJsonStr(buildMultipartPreviewData(c, form)), nil
}

func buildMultipartPreviewData(c *gin.Context, form *multipart.Form) map[string]any {
	previewBody := make(map[string]any, len(form.Value)+len(form.File))

	for key, values := range form.Value {
		if len(values) == 1 {
			previewBody[key] = values[0]
		} else {
			previewBody[key] = values
		}
	}

	for fieldName, fileHeaders := range form.File {
		fileNames := make([]string, 0, len(fileHeaders))
		for _, fileHeader := range fileHeaders {
			fileNames = append(fileNames, fileHeader.Filename)
		}
		if len(fileNames) == 1 {
			previewBody[fieldName] = fileNames[0]
		} else {
			previewBody[fieldName] = fileNames
		}
	}

	return previewBody
}

func decodeUnicodeForLog(body []byte, contentType string) string {
	if !strings.HasPrefix(contentType, gin.MIMEJSON) || len(body) == 0 {
		return string(body)
	}

	var payload any
	if err := common.Unmarshal(body, &payload); err != nil {
		return string(body)
	}
	pretty, err := common.Marshal(payload)
	if err != nil {
		return string(body)
	}
	return string(pretty)
}

func sanitizeRequestBodyForLog(body []byte, contentType string) string {
	decoded := decodeUnicodeForLog(body, contentType)
	if decoded == "" {
		return decoded
	}
	if strings.HasPrefix(contentType, gin.MIMEJSON) {
		return common.SanitizeJSONStringForLog([]byte(decoded))
	}
	return common.SanitizeStringForLog(decoded)
}
