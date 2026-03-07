package xiezhi

import (
	"crypto/rsa"
	"crypto/sha256"
	"hash"
)

type Rsa struct {
	pubKey []byte
	priKey []byte
	pri    *rsa.PrivateKey
	pub    *rsa.PublicKey
	hash   hash.Hash
}

func NewRsa(pub []byte, pri ...[]byte) (h Handler, err error) {
	var r = &Rsa{pubKey: pub}
	if len(pri) > 0 {
		r.priKey = pri[0]
	}
	err = r.Init()
	h = r
	return
}

func (r *Rsa) Init() (err error) {

	r.hash = sha256.New()
	pub, err := loadRsaPub(r.pubKey)
	if err != nil {
		return
	}
	r.pub = pub

	if len(r.priKey) == 0 {
		return
	}
	pri, err := loadRsaPri(r.priKey)
	if err != nil {
		return
	}
	r.pri = pri
	return
}

func (r *Rsa) Verify(payload []byte, signature []byte) (err error) {
	// input -> hash, digest -> 解密 -> 对比两个签名
	r.hash.Reset()
	r.hash.Write(payload)
	hashed := r.hash.Sum(nil)
	err = rsaVerify(hashed, r.pub, signature)
	if err != nil {
		return
	}
	return
}

func (r *Rsa) Sign(input []byte) (digest []byte, err error) {
	// 明文->hash->加密
	r.hash.Reset()
	r.hash.Write(input)
	bs := r.hash.Sum(nil)
	digest, err = rsaSign(bs, r.pri)
	if err != nil {
		return
	}

	return
}
