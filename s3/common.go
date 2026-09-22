package s3

import (
	"errors"
	"time"
)

const (
	Fs    ProviderCategory = "fs"    // 本地文件系统
	S3    ProviderCategory = "s3"    // AWS S3
	OSS   ProviderCategory = "oss"   // 阿里云 OSS（S3 兼容）
	COS   ProviderCategory = "cos"   // 腾讯云 COS
	OBS   ProviderCategory = "obs"   // 华为云 OBS
	TOS   ProviderCategory = "tos"   // 火山引擎 TOS
	JDOS  ProviderCategory = "jdos"  // 京东云
	KODOS ProviderCategory = "kodos" // 七牛云 Kodo
	BOS   ProviderCategory = "bos"   // 百度云 BOS
)

var SupportedCategory = make(map[ProviderCategory]EndpointTpl)

func init() {
	entries := []struct {
		c ProviderCategory
		t EndpointTpl
	}{
		{Fs, "{region}"},
		{S3, "https://s3.{region}.amazonaws.com"},
		{OSS, "https://s3.oss-{region}.aliyuncs.com"},
		{COS, "https://cos.{region}.myqcloud.com"},
		{OBS, "https://obs.{region}.myhuaweicloud.com"},
		{TOS, "https://tos-s3-{region}.volces.com"},
		{JDOS, "https://s3.{region}.jdcloud-oss.com"},
		{KODOS, "https://s3-{region}.qiniucs.com"},
		{BOS, "https://s3.{region}.bcebos.com"},
	}
	for _, e := range entries {
		SupportedCategory[e.c] = e.t
	}
}

var (
	ErrProviderNotFound    = errors.New("provider not exist")
	ErrBuilderNil          = errors.New("builder is nil")
	ErrInvalidObjectKey    = errors.New("invalid object key")
	ErrFsRootRequired      = errors.New("fs region(root dir) required")
	ErrFsRootNotAllowed    = errors.New("fs root not registered")
	ErrS3ConfigNotFound    = errors.New("s3 provider config not found")
	ErrS3BucketRequired    = errors.New("s3 bucket required")
	ErrUnsupportedCategory = errors.New("unsupported provider category")
	ErrReadFnNil           = errors.New("read callback is nil")
)

var (
	// DefaultProviderExpired Provider 缓存过期时间（供 Watch 刷新）
	DefaultProviderExpired = time.Hour
	// DefaultWatchInterval 默认轮询间隔
	DefaultWatchInterval = 5 * time.Minute
)

// IsObjectCategory 是否为对象存储类（非本地 fs）。
func IsObjectCategory(c ProviderCategory) bool {
	switch c {
	case S3, OSS, COS, OBS, TOS, JDOS, KODOS, BOS:
		return true
	default:
		return false
	}
}