// Package s3 提供文件网关：按 ProviderCategory + Region 缓存 Provider 实例，
// 通过 ProviderFactory 构建实例，Gateway.Provider 获取客户端执行文件操作。//
// 用法概要：
//
//	gw, _ := s3.NewGateway(myFactory)
//	gw.StartWatch(ctx)
//	p, _ := gw.Provider(s3.Fs, s3.Region("/data/uploads"))
//	p.Save(ctx, "path/to/file", reader)
//	_ = p.Open(ctx, "path/to/file", func(r io.Reader) error { _, err := io.ReadAll(r); return err })
//
// 内置实现：
//   - ProviderFs：本地文件系统，Region 为根目录
//   - ProviderS3：S3 兼容对象存储（minio-go 客户端）
//
// 可选 Registry 提供参考版 Factory（RegisterFsRoot / RegisterObject + Factory()）。
package s3
