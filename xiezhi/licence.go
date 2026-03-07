package xiezhi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Licence struct {
	Signature []byte
	Payload   []byte
}

func (l *Licence) Marshall() (s string) {
	d := base64.StdEncoding.EncodeToString(l.Signature)
	p := base64.StdEncoding.EncodeToString(l.Payload)
	s = fmt.Sprintf("%s.%s", p, d)
	return
}

func (l *Licence) Unmarshall(input string) (err error) {
	inputs := strings.Split(input, ".")
	if len(inputs) != 2 {
		err = errors.New("invalid input")
		return
	}
	l.Payload, err = base64.StdEncoding.DecodeString(inputs[0])
	if err != nil {
		return
	}
	l.Signature, err = base64.StdEncoding.DecodeString(inputs[1])
	if err != nil {
		return
	}
	return
}

// Verify 验证签名前需要配置payload和signature
func (l *Licence) Verify(h Handler) (err error) {
	if len(l.Signature) == 0 {
		err = errors.New("invalid signature")
		return
	}
	if len(l.Payload) == 0 {
		err = errors.New("invalid payload")
		return
	}
	err = h.Verify(l.Payload, l.Signature)
	return
}

// Sign 计算签名前需要配置payload
func (l *Licence) Sign(h Handler) (err error) {
	if len(l.Payload) == 0 {
		err = errors.New("empty payload")
		return
	}
	sig, err := h.Sign(l.Payload)
	if err != nil {
		return
	}
	l.Signature = sig
	return
}

// SignWithObject 可序列化的payload
func (l *Licence) SignWithObject(h Handler, payload any) (err error) {
	bs, err := json.Marshal(payload)
	if err != nil {
		return
	}
	l.Payload = bs
	return l.Sign(h)
}
