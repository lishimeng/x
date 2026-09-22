package redisstream

// DefaultMaxLen 单流默认近似上限条数（Publish 时 MAXLEN ~）。
const DefaultMaxLen int64 = 10000

// Option 配置 Bus。
type Option func(*Bus)

// WithMaxLen 设置单流近似上限；0 关闭裁剪。未设置时用 DefaultMaxLen。
func WithMaxLen(n int64) Option {
	return func(b *Bus) {
		if b == nil {
			return
		}
		b.maxLen = n
	}
}
