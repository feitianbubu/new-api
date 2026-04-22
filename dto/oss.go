package dto

// OSSUploadRequest 火山云TOS文件上传请求
type OSSUploadRequest struct {
	BucketName  string            `json:"bucket_name" binding:"required" example:"my-bucket"`       // 存储桶名称
	ObjectKey   string            `json:"object_key" binding:"required" example:"images/photo.jpg"` // 对象键（文件路径）
	ContentType string            `json:"content_type,omitempty" example:"image/jpeg"`              // 文件内容类型
	ACL         string            `json:"acl,omitempty" example:"private"`                          // 访问控制列表
	Metadata    map[string]string `json:"metadata,omitempty"`                                       // 自定义元数据
}

// OSSUploadResponse 火山云TOS文件上传响应
type OSSUploadResponse struct {
	ETag         string `json:"etag" example:"\"d41d8cd98f00b204e9800998ecf8427e\""`                             // 文件ETag
	VersionID    string `json:"version_id,omitempty" example:"null"`                                             // 版本ID（如果启用版本控制）
	ObjectKey    string `json:"object_key" example:"images/photo.jpg"`                                           // 对象键
	BucketName   string `json:"bucket_name" example:"my-bucket"`                                                 // 存储桶名称
	Location     string `json:"location" example:"https://my-bucket.tos-cn-beijing.volces.com/images/photo.jpg"` // 文件访问URL
	Size         int64  `json:"size" example:"1024"`                                                             // 文件大小（字节）
	LastModified string `json:"last_modified" example:"2024-01-15T10:30:00Z"`                                    // 最后修改时间
}

// OSSUploadError 火山云TOS文件上传错误响应
type OSSUploadError struct {
	Code      string `json:"code" example:"NoSuchBucket"`                           // 错误代码
	Message   string `json:"message" example:"The specified bucket does not exist"` // 错误消息
	RequestID string `json:"request_id" example:"16A4A75A92C8C4E3"`                 // 请求ID
	HostID    string `json:"host_id" example:"tos-cn-beijing.volces.com"`           // 主机ID
}

// OSSBucketInfo 存储桶信息
type OSSBucketInfo struct {
	Name         string `json:"name" example:"my-bucket"`                     // 存储桶名称
	CreationDate string `json:"creation_date" example:"2024-01-15T10:30:00Z"` // 创建时间
	Location     string `json:"location" example:"cn-beijing"`                // 存储桶所在区域
}

// OSSObjectInfo 对象信息
type OSSObjectInfo struct {
	Key          string            `json:"key" example:"images/photo.jpg"`                      // 对象键
	ETag         string            `json:"etag" example:"\"d41d8cd98f00b204e9800998ecf8427e\""` // 对象ETag
	Size         int64             `json:"size" example:"1024"`                                 // 对象大小
	LastModified string            `json:"last_modified" example:"2024-01-15T10:30:00Z"`        // 最后修改时间
	StorageClass string            `json:"storage_class" example:"STANDARD"`                    // 存储类型
	Metadata     map[string]string `json:"metadata,omitempty"`                                  // 自定义元数据
}
