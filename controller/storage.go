package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/storage"

	"github.com/gin-gonic/gin"
)

// Upload
// @Summary 上传文件
// @Description 上传文件到对象存储服务，兼容 OpenAI Files API 格式
// @Description 基于OpenAI Files API规范，支持文件上传、列出、获取信息、删除等操作
// @Description - 参考文档：https://platform.openai.com/docs/api-reference/files
// @Tags Files
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "用户认证令牌 (Bearer sk-xxxx)"
// @Param file formData file true "要上传的文件" example(@file.jpg)
// @Param purpose formData string false "文件用途" example(user_data)
// @Param expires_after formData string false "过期策略（JSON格式），默认14天" example({"anchor":"created_at","seconds":86400})
// @Success 200 {object} dto.FileObject "上传成功"
// @Failure 400 {object} dto.FileError "请求参数错误"
// @Failure 401 {object} dto.FileError "未授权"
// @Failure 403 {object} dto.FileError "无权限"
// @Failure 404 {object} dto.FileError "存储桶不存在"
// @Failure 500 {object} dto.FileError "服务器内部错误"
// @Router /v1/files [post]
func Upload(c *gin.Context) {
	// Get user info
	userId := c.GetInt("id")
	if userId == 0 {
		FailFileError(c, http.StatusUnauthorized, "authentication_error", "invalid_token", "user not authenticated")
		return
	}

	tokenKey := c.GetString("token_key")
	userGroup := c.GetString("group")
	usingGroup := c.GetString("using_group")
	if usingGroup == "" {
		usingGroup = "default"
	}

	// Parse form data
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		FailFileError(c, http.StatusBadRequest, "invalid_request_error", "invalid_file", "failed to get uploaded file: "+err.Error())
		return
	}
	defer file.Close()

	// Get form parameters
	purpose := c.PostForm("purpose")
	if purpose == "" {
		purpose = "user_data" // Default purpose
	}

	// Parse expires_after parameter (OpenAI format) with priority
	var expiresAfter *storage.ExpiresAfter

	expiresAfterStr := c.PostForm("expires_after")
	if expiresAfterStr != "" {
		var expiresAfterData dto.ExpiresAfter
		if err := json.Unmarshal([]byte(expiresAfterStr), &expiresAfterData); err != nil {
			FailFileError(c, http.StatusBadRequest, "invalid_request_error", "invalid_expires_after", "invalid expires_after format: "+err.Error())
			return
		}

		// Validate expires_after parameters
		if expiresAfterData.Anchor != "created_at" {
			FailFileError(c, http.StatusBadRequest, "invalid_request_error", "invalid_anchor", "invalid expires_after anchor: only 'created_at' is supported")
			return
		}

		//if expiresAfterData.Seconds < 3600 || expiresAfterData.Seconds > 2592000 {
		//	Fail(c, "invalid expires_after seconds: must be between 3600 (1 hour) and 2592000 (30 days)")
		//	return
		//}

		expiresAfter = &storage.ExpiresAfter{
			Anchor:  expiresAfterData.Anchor,
			Seconds: expiresAfterData.Seconds,
		}

	} else {
		// Default to 14 days if no expires_after provided
		expiresAfter = &storage.ExpiresAfter{
			Anchor:  "created_at",
			Seconds: storage.DefaultStorageSeconds,
		}
	}

	// Set content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Create storage instance
	storageInstance, err := storage.NewStorageFromEnv()
	if err != nil {
		logger.LogError(c, "failed to create storage instance: "+err.Error())
		FailFileError(c, http.StatusInternalServerError, "internal_error", "storage_init_failed", "failed to initialize storage")
		return
	}
	defer storageInstance.Close()

	// Upload file
	fileObj, err := storageInstance.UploadFile(c, file, header.Size, storage.UploadOptions{
		Filename:     header.Filename,
		ContentType:  contentType,
		Purpose:      purpose,
		UserID:       userId,
		ExpiresAfter: expiresAfter,
	})
	if err != nil {
		logger.LogError(c, "failed to upload file: "+err.Error())
		FailFileError(c, http.StatusInternalServerError, "internal_error", "upload_failed", "failed to upload file to storage")
		return
	}

	var usage *dto.Usage
	fileSizeMB := float64(fileObj.Bytes) / (1024 * 1024) // Convert to MB
	promptTokens := int(fileObj.Bytes)
	// Calculate equivalent storage days for quota calculation (minimum 1 day)
	storageDaysEquiv := int64(math.Ceil(float64(expiresAfter.Seconds) / 86400.0))
	if storageDaysEquiv == 0 {
		storageDaysEquiv = 1
	}
	totalTokens := int(math.Ceil(float64(fileObj.Bytes*storageDaysEquiv) * storage.ModelRatio))

	groupRatio := ratio_setting.GetGroupRatio(usingGroup)

	if userGroupRatio, ok := ratio_setting.GetGroupGroupRatio(userGroup, usingGroup); ok {
		groupRatio = userGroupRatio
	}

	quota := int(float64(totalTokens) * groupRatio)

	if quota > 0 {
		// Check user quota
		userQuota, err := model.GetUserQuota(userId, false)
		if err != nil {
			logger.LogError(c, "failed to get user quota: "+err.Error())
			FailFileError(c, http.StatusInternalServerError, "internal_error", "quota_check_failed", "failed to get user quota")
			return
		}

		if userQuota < quota {
			FailFileError(c, http.StatusForbidden, "quota_exceeded", "insufficient_quota", "insufficient quota for file storage")
			return
		}

		token, err := model.ValidateUserToken(tokenKey)
		if err != nil {
			logger.LogError(c, "failed to get token: "+err.Error())
			FailFileError(c, http.StatusInternalServerError, "internal_error", "token_validation_failed", "failed to get token")
			return
		}

		if !token.UnlimitedQuota && token.RemainQuota < quota {
			FailFileError(c, http.StatusForbidden, "quota_exceeded", "insufficient_token_quota", "insufficient token quota for file storage")
			return
		}

		var userSetting dto.UserSetting
		if us, ok := common.GetContextKeyType[dto.UserSetting](c, constant.ContextKeyUserSetting); ok {
			userSetting = us
		}
		relayInfo := &relaycommon.RelayInfo{
			UserId:         userId,
			UserEmail:      common.GetContextKeyString(c, constant.ContextKeyUserEmail),
			UserSetting:    userSetting,
			UserQuota:      userQuota,
			TokenId:        token.Id,
			TokenKey:       tokenKey,
			TokenUnlimited: token.UnlimitedQuota,
		}
		err = service.PostConsumeQuota(relayInfo, quota, 0, true)
		if err != nil {
			logger.LogError(c, "failed to post consume quota: "+err.Error())
			FailFileError(c, http.StatusInternalServerError, "internal_error", "quota_deduction_failed", "failed to decrease user quota")
			return
		}

		// Update usage statistics
		model.UpdateUserUsedQuotaAndRequestCount(userId, quota)

		// Record consumption log
		tokenName := c.GetString("token_name")
		logContent := fmt.Sprintf("模型倍率 %.2f，补全倍率 %.2f，分组倍率 %.2f", storage.ModelRatio, 1.0, groupRatio)
		other := map[string]interface{}{
			"file_id":            fileObj.ID,
			"file_size_mb":       fileSizeMB,
			"expires_after":      expiresAfter,
			"storage_days_equiv": storageDaysEquiv, // For quota calculation only
			"model_ratio":        storage.ModelRatio,
			"group_ratio":        groupRatio,
			"promptTokens":       promptTokens,
			"totalTokens":        totalTokens,
			"quota":              quota,
		}
		model.RecordConsumeLog(c, userId, model.RecordConsumeLogParams{
			PromptTokens:     promptTokens,
			CompletionTokens: 0,
			ModelName:        storageInstance.GetModelName(),
			TokenName:        tokenName,
			Quota:            quota,
			Content:          logContent,
			TokenId:          token.Id,
			//UserQuota:        userQuota,
			Group: usingGroup,
			Other: other,
		})
	}

	// Create usage object
	usage = &dto.Usage{
		TotalTokens: promptTokens,
	}

	// Create OpenAI Files API compatible response
	response := dto.FileObject{
		ID:          fileObj.ID,
		Object:      "file",
		Bytes:       fileObj.Bytes,
		CreatedAt:   fileObj.CreatedAt.Unix(),
		ExpiresAt:   fileObj.ExpiresAt.Unix(),
		Filename:    fileObj.Filename,
		ContentType: fileObj.ContentType,
		Purpose:     fileObj.Purpose,
		Usage:       usage,
	}

	Success(c, response)
}

// ListFiles
// @Summary 列出文件
// @Description 列出用户上传的文件列表，兼容 OpenAI Files API 格式
// @Description 支持两种模式：1) 按用户列出（默认） 2) 按对象路径前缀列出（传入prefix参数）
// @Tags Files
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string false "用户认证令牌 (Bearer sk-xxxx)，按用户列出文件时必需"
// @Param limit query int false "返回文件数量限制" example(20)
// @Param after query string false "分页游标" example(file-d0cebe01d690822d3977d1809e694788)
// @Param purpose query string false "文件用途过滤" example(user_data)
// @Param prefix query string false "对象路径前缀，用于列出特定目录下的文件（如日志）" example(logs/20260103/abc123/)
// @Success 200 {object} dto.FileListResponse "文件列表"
// @Failure 400 {object} dto.FileError "请求参数错误"
// @Failure 401 {object} dto.FileError "未授权"
// @Failure 500 {object} dto.FileError "服务器内部错误"
// @Router /v1/files [get]
func ListFiles(c *gin.Context) {
	// Get query parameters
	limitStr := c.Query("limit")
	after := c.Query("after")
	purpose := c.Query("purpose")
	prefix := c.Query("prefix")

	// Parse limit
	limit := 20 // Default limit
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// 如果有 prefix 参数，则按 prefix 查询（不需要认证）
	// 否则按 userId 查询（需要认证）
	var listOpts storage.ListOptions
	if prefix != "" {
		// 按 prefix 模式
		listOpts = storage.ListOptions{
			Prefix:  prefix,
			Limit:   limit,
			After:   after,
			Purpose: purpose,
		}
	} else {
		// 按 userId 模式（需要认证）
		userId := c.GetInt("id")
		if userId == 0 {
			FailFileError(c, http.StatusUnauthorized, "authentication_error", "invalid_token", "user not authenticated")
			return
		}
		listOpts = storage.ListOptions{
			UserID:  userId,
			Limit:   limit,
			After:   after,
			Purpose: purpose,
		}
	}

	// Create storage instance
	storageInstance, err := storage.NewStorageFromEnv()
	if err != nil {
		logger.LogError(c, "failed to create storage instance: "+err.Error())
		FailFileError(c, http.StatusInternalServerError, "internal_error", "storage_init_failed", "failed to initialize storage")
		return
	}
	defer storageInstance.Close()

	// List files
	result, err := storageInstance.ListFiles(c, listOpts)
	if err != nil {
		logger.LogError(c, "failed to list files: "+err.Error())
		FailFileError(c, http.StatusInternalServerError, "internal_error", "list_files_failed", "failed to list files")
		return
	}

	// Convert to DTO format
	var files []dto.FileObject
	for _, file := range result.Files {
		files = append(files, dto.FileObject{
			ID:          file.ID,
			Object:      "file",
			Bytes:       file.Bytes,
			CreatedAt:   file.CreatedAt.Unix(),
			ExpiresAt:   file.ExpiresAt.Unix(),
			Filename:    file.Filename,
			ContentType: file.ContentType,
			Purpose:     file.Purpose,
		})
	}

	// Create response
	response := dto.FileListResponse{
		Object:  "list",
		Data:    files,
		HasMore: result.HasMore,
	}

	Success(c, response)
}

// GetFileInfo
// @Summary 文件信息
// @Description 获取指定文件的详细信息，兼容 OpenAI Files API 格式
// @Tags Files
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "用户认证令牌 (Bearer sk-xxxx)"
// @Param file_id path string true "文件ID" example(file-d0cebe01d690822d3977d1809e694788)
// @Success 200 {object} dto.FileObject "文件信息"
// @Failure 400 {object} dto.FileError "请求参数错误"
// @Failure 401 {object} dto.FileError "未授权"
// @Failure 404 {object} dto.FileError "文件不存在"
// @Failure 500 {object} dto.FileError "服务器内部错误"
// @Router /v1/files/{file_id} [get]
func GetFileInfo(c *gin.Context) {
	// Get user info
	userId := c.GetInt("id")
	if userId == 0 {
		FailFileError(c, http.StatusUnauthorized, "authentication_error", "invalid_token", "user not authenticated")
		return
	}

	// Get file ID from path
	fileID := c.Param("file_id")
	if fileID == "" {
		FailFileError(c, http.StatusBadRequest, "invalid_request_error", "missing_file_id", "file_id is required")
		return
	}

	// Create storage instance
	storageInstance, err := storage.NewStorageFromEnv()
	if err != nil {
		logger.LogError(c, "failed to create storage instance: "+err.Error())
		FailFileError(c, http.StatusInternalServerError, "internal_error", "storage_init_failed", "failed to initialize storage")
		return
	}
	defer storageInstance.Close()

	// Get file info
	fileObj, err := storageInstance.GetFileInfo(c, fileID)
	if err != nil {
		logger.LogError(c, "failed to get file info: "+err.Error())
		FailFileError(c, http.StatusNotFound, "not_found_error", "file_not_found", "failed to get file information")
		return
	}

	// Create response
	response := dto.FileObject{
		ID:          fileObj.ID,
		Object:      "file",
		Bytes:       fileObj.Bytes,
		CreatedAt:   fileObj.CreatedAt.Unix(),
		ExpiresAt:   fileObj.ExpiresAt.Unix(),
		Filename:    fileObj.Filename,
		ContentType: fileObj.ContentType,
		Purpose:     fileObj.Purpose,
	}

	Success(c, response)
}

// DeleteFile
// @Summary 删除文件
// @Description 删除指定文件，兼容 OpenAI Files API 格式
// @Tags Files
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "用户认证令牌 (Bearer sk-xxxx)"
// @Param file_id path string true "文件ID" example(file-d0cebe01d690822d3977d1809e694788)
// @Success 200 {object} dto.FileDeleteResponse "删除成功"
// @Failure 400 {object} dto.FileError "请求参数错误"
// @Failure 401 {object} dto.FileError "未授权"
// @Failure 404 {object} dto.FileError "文件不存在"
// @Failure 500 {object} dto.FileError "服务器内部错误"
// @Router /v1/files/{file_id} [delete]
func DeleteFile(c *gin.Context) {
	// Get user info
	userId := c.GetInt("id")
	if userId == 0 {
		FailFileError(c, http.StatusUnauthorized, "authentication_error", "invalid_token", "user not authenticated")
		return
	}

	// Get file ID from path
	fileID := c.Param("file_id")
	if fileID == "" {
		FailFileError(c, http.StatusBadRequest, "invalid_request_error", "missing_file_id", "file_id is required")
		return
	}

	// Create storage instance
	storageInstance, err := storage.NewStorageFromEnv()
	if err != nil {
		logger.LogError(c, "failed to create storage instance: "+err.Error())
		FailFileError(c, http.StatusInternalServerError, "internal_error", "storage_init_failed", "failed to initialize storage")
		return
	}
	defer storageInstance.Close()

	// Delete file
	err = storageInstance.DeleteFile(c, fileID)
	if err != nil {
		logger.LogError(c, "failed to delete file: "+err.Error())
		FailFileError(c, http.StatusNotFound, "not_found_error", "file_not_found", "failed to delete file")
		return
	}

	// Create response
	response := dto.FileDeleteResponse{
		ID:      fileID,
		Object:  "file",
		Deleted: true,
	}

	Success(c, response)
}

// GetFileContent
// @Summary 文件内容
// @Description 获取指定文件的内容，兼容 OpenAI Files API 格式
// @Tags Files
// @Accept json
// @Produce application/octet-stream
// @Security BearerAuth
// @Param Authorization header string true "用户认证令牌 (Bearer sk-xxxx)"
// @Param file_id path string true "文件ID" example(file-d0cebe01d690822d3977d1809e694788)
// @Success 200 {file} file "文件内容"
// @Failure 400 {object} dto.FileError "请求参数错误"
// @Failure 401 {object} dto.FileError "未授权"
// @Failure 404 {object} dto.FileError "文件不存在"
// @Failure 500 {object} dto.FileError "服务器内部错误"
// @Router /v1/files/{file_id}/content [get]
func GetFileContent(c *gin.Context) {
	fileID := c.Param("file_id")
	if fileID == "" {
		FailFileError(c, http.StatusBadRequest, "invalid_request_error", "missing_file_id", "file_id is required")
		return
	}

	rangeHeader := c.GetHeader("Range")

	storageInstance, err := storage.NewStorageFromEnv()
	if err != nil {
		logger.LogError(c, "failed to create storage instance: "+err.Error())
		FailFileError(c, http.StatusInternalServerError, "internal_error", "storage_init_failed", "failed to initialize storage")
		return
	}
	defer storageInstance.Close()

	fileContent, err := storageInstance.GetFileContent(c, fileID)
	if err != nil {
		logger.LogError(c, "failed to get file content: "+err.Error())
		FailFileError(c, http.StatusNotFound, "not_found_error", "file_not_found", "failed to get file content")
		return
	}
	defer fileContent.Content.Close()

	c.Header("Content-Type", fileContent.ContentType)
	c.Header("Accept-Ranges", "bytes")

	if rangeHeader != "" && fileContent.RangeEnd > 0 {
		c.Status(http.StatusPartialContent)
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", fileContent.RangeStart, fileContent.RangeEnd, fileContent.TotalLength))
		c.Header("Content-Length", strconv.FormatInt(fileContent.ContentLength, 10))
	} else {
		c.Header("Content-Length", strconv.FormatInt(fileContent.ContentLength, 10))
	}

	if fileContent.Filename != "" {
		c.Header("Content-Disposition", fmt.Sprintf("filename=\"%s\"", fileContent.Filename))
	}

	_, err = io.Copy(c.Writer, fileContent.Content)
	if err != nil {
		logger.LogError(c, "failed to stream file content: "+err.Error())
		return
	}
}
