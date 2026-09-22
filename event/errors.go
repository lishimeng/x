package event

import (
	"errors"
	"fmt"
)

// NoAckError 通信/基础设施失败：订阅方不得 ACK，保留 pending 以便重试。
// 与业务异常分离——业务异常只用于打印/写日志（return nil，或 return 普通 err 由框架打日志后 ACK）。
type NoAckError struct {
	Err error
}

func (e *NoAckError) Error() string {
	if e == nil || e.Err == nil {
		return "event: no-ack"
	}
	return e.Err.Error()
}

func (e *NoAckError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NoAck 包装通信层异常，控制 ACK（不确认、可重试）。err 为 nil 时返回 nil。
func NoAck(err error) error {
	if err == nil {
		return nil
	}
	var na *NoAckError
	if errors.As(err, &na) {
		return err
	}
	return &NoAckError{Err: err}
}

// NoAckf 同 NoAck，带格式化。
func NoAckf(format string, args ...any) error {
	return NoAck(fmt.Errorf(format, args...))
}

// IsNoAck 是否为控制 ACK 的通信异常（含 wrap 链）。
func IsNoAck(err error) bool {
	var na *NoAckError
	return errors.As(err, &na)
}

// Deprecated: 使用 NoAck。
func NonIgnorable(err error) error { return NoAck(err) }

// Deprecated: 使用 NoAckf。
func NonIgnorableF(format string, args ...any) error { return NoAckf(format, args...) }

// Deprecated: 使用 IsNoAck。
func IsNonIgnorable(err error) bool { return IsNoAck(err) }

// Deprecated: 使用 NoAckError。
type NonIgnorableError = NoAckError
