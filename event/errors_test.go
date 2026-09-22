package event

import (
	"errors"
	"fmt"
	"testing"
)

func TestDebugFlag(t *testing.T) {
	SetDebug(true)
	t.Log(Debug)
	SetDebug(false)
	t.Log(Debug)
}

func TestNoAck(t *testing.T) {
	if NoAck(nil) != nil {
		t.Fatal("nil")
	}
	base := errors.New("redis down")
	wrapped := NoAck(base)
	if !IsNoAck(wrapped) {
		t.Fatal("expected no-ack")
	}
	if !errors.Is(wrapped, base) {
		t.Fatal("unwrap")
	}
	if NoAck(wrapped) != wrapped {
		t.Fatal("double wrap")
	}
	biz := fmt.Errorf("套餐不存在")
	if IsNoAck(biz) {
		t.Fatal("plain biz err must not hold ack")
	}
	// 兼容旧名
	if !IsNonIgnorable(NonIgnorable(base)) {
		t.Fatal("alias")
	}
}
