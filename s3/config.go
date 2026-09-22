package s3

// ProviderConfig 供应器配置（字段包内可见，通过构造函数创建）
type ProviderConfig struct {
	Auth
	endpoint Endpoint
	region   Region
	category ProviderCategory
	bucket   string // 对象存储 bucket；fs 忽略
}

func (c ProviderConfig) Endpoint() Endpoint       { return c.endpoint }
func (c ProviderConfig) Region() Region           { return c.region }
func (c ProviderConfig) Category() ProviderCategory { return c.category }
func (c ProviderConfig) Bucket() string           { return c.bucket }

// NewFsProviderConfig region 为本地根目录（相对或绝对路径）
func NewFsProviderConfig(root Region) ProviderConfig {
	return ProviderConfig{
		category: Fs,
		region:   root,
		endpoint: Endpoint(SupportedCategory[Fs].Format(root)),
	}
}

// NewObjectProviderConfig 创建 S3 兼容对象存储配置
func NewObjectProviderConfig(category ProviderCategory, region Region, bucket string, auth Auth) (ProviderConfig, error) {
	if !IsObjectCategory(category) {
		return ProviderConfig{}, ErrUnsupportedCategory
	}
	if bucket == "" {
		return ProviderConfig{}, ErrS3BucketRequired
	}
	tpl, ok := SupportedCategory[category]
	if !ok {
		return ProviderConfig{}, ErrUnsupportedCategory
	}
	return ProviderConfig{
		Auth:     auth,
		endpoint: Endpoint(tpl.Format(region)),
		region:   region,
		category: category,
		bucket:   bucket,
	}, nil
}
