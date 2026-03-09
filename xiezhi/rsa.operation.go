package xiezhi

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

func loadRsaPub(data []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pub.(*rsa.PublicKey), nil
}

func isPKCS8Header(h string) bool { // 兼容pkcs8与pkcs1的header(pkcs1只限RSA)
	return h == "PRIVATE KEY" || h == "RSA PRIVATE KEY"
}

func loadRsaPri(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil || !isPKCS8Header(block.Type) {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKCS#8 private key: %w", err)
	}

	rsaPriv, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}
	return rsaPriv, nil
}

func rsaVerify(hashed []byte, pub *rsa.PublicKey, signature []byte) error {
	return rsa.VerifyPSS(pub, crypto.SHA256, hashed, signature, &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthAuto,
	})
}

func rsaSign(hashed []byte, priv *rsa.PrivateKey) ([]byte, error) {
	return rsa.SignPSS(rand.Reader, priv, crypto.SHA256, hashed, &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthAuto,
	})
}
