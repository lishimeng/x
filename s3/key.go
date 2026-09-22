package s3

import (
	"path/filepath"
	"strings"
)

// normalizeObjectKey 对象键：相对路径、禁止 .. 与绝对路径
func normalizeObjectKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", ErrInvalidObjectKey
	}
	key = filepath.ToSlash(filepath.Clean(key))
	if key == "." || strings.HasPrefix(key, "../") || strings.Contains(key, "/../") {
		return "", ErrInvalidObjectKey
	}
	if filepath.IsAbs(key) || strings.HasPrefix(key, "/") {
		return "", ErrInvalidObjectKey
	}
	return key, nil
}
