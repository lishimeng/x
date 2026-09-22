package s3

import (
	"context"
	"time"
)

// PresignPostResult COS/S3 POST 表单直传签名结果（供 wx.uploadFile + formData）
type PresignPostResult struct {
	UploadURL string            `json:"uploadUrl"`
	FormData  map[string]string `json:"formData"`
}

// PresignProvider 支持预签名上传/下载的对象存储后端
type PresignProvider interface {
	PresignPost(ctx context.Context, key string, expiry time.Duration) (PresignPostResult, error)
	PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error)
}
