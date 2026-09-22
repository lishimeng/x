package s3

import (
	"context"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// s3Client S3 兼容客户端（minio-go）
type s3Client interface {
	PutObject(ctx context.Context, bucket, object string, reader io.Reader, size int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	GetObject(ctx context.Context, bucket, object string, opts minio.GetObjectOptions) (*minio.Object, error)
	RemoveObject(ctx context.Context, bucket, object string, opts minio.RemoveObjectOptions) error
	PresignedPostPolicy(ctx context.Context, policy *minio.PostPolicy) (u *url.URL, formData map[string]string, err error)
	PresignedGetObject(ctx context.Context, bucket, object string, expires time.Duration, reqParams url.Values) (*url.URL, error)
}

// NewProviderS3 创建 S3 兼容对象存储 provider
func NewProviderS3(cfg ProviderConfig) (*ProviderS3, error) {
	if !IsObjectCategory(cfg.category) {
		return nil, ErrUnsupportedCategory
	}
	if cfg.bucket == "" {
		return nil, ErrS3BucketRequired
	}
	client, err := newMinioClient(cfg)
	if err != nil {
		return nil, err
	}
	return &ProviderS3{
		BaseProvider: BaseProvider{config: cfg},
		client:       client,
	}, nil
}

type ProviderS3 struct {
	BaseProvider
	client s3Client
}

func newMinioClient(cfg ProviderConfig) (*minio.Client, error) {
	ep := strings.TrimSpace(string(cfg.endpoint))
	useSSL := strings.HasPrefix(ep, "https://")
	host := strings.TrimPrefix(strings.TrimPrefix(ep, "https://"), "http://")
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: useSSL,
		Region: string(cfg.region),
	}
	if cfg.category == COS {
		opts.BucketLookup = minio.BucketLookupDNS
	}
	return minio.New(host, opts)
}

func (p *ProviderS3) Save(ctx context.Context, key string, r io.Reader) (written int64, err error) {
	if err = ctx.Err(); err != nil {
		return
	}
	rel, err := normalizeObjectKey(key)
	if err != nil {
		return
	}
	info, err := p.client.PutObject(ctx, p.config.bucket, rel, r, -1, minio.PutObjectOptions{})
	if err != nil {
		return
	}
	return info.Size, nil
}

func (p *ProviderS3) Open(ctx context.Context, key string, fn ReadFn) error {
	if fn == nil {
		return ErrReadFnNil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	rel, err := normalizeObjectKey(key)
	if err != nil {
		return err
	}
	obj, err := p.client.GetObject(ctx, p.config.bucket, rel, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = obj.Close() }()
	return fn(obj)
}

func (p *ProviderS3) Remove(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	rel, err := normalizeObjectKey(key)
	if err != nil {
		return err
	}
	return p.client.RemoveObject(ctx, p.config.bucket, rel, minio.RemoveObjectOptions{})
}

func (p *ProviderS3) PresignPost(ctx context.Context, key string, expiry time.Duration) (PresignPostResult, error) {
	if err := ctx.Err(); err != nil {
		return PresignPostResult{}, err
	}
	rel, err := normalizeObjectKey(key)
	if err != nil {
		return PresignPostResult{}, err
	}
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}
	policy := minio.NewPostPolicy()
	if err = policy.SetBucket(p.config.bucket); err != nil {
		return PresignPostResult{}, err
	}
	if err = policy.SetKey(rel); err != nil {
		return PresignPostResult{}, err
	}
	if err = policy.SetExpires(time.Now().UTC().Add(expiry)); err != nil {
		return PresignPostResult{}, err
	}
	u, formData, err := p.client.PresignedPostPolicy(ctx, policy)
	if err != nil {
		return PresignPostResult{}, err
	}
	return PresignPostResult{UploadURL: u.String(), FormData: formData}, nil
}

func (p *ProviderS3) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	rel, err := normalizeObjectKey(key)
	if err != nil {
		return "", err
	}
	if expiry <= 0 {
		expiry = time.Hour
	}
	u, err := p.client.PresignedGetObject(ctx, p.config.bucket, rel, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
