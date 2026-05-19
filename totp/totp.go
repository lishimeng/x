package totp

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// 兼容 Google Authenticator / Microsoft Authenticator 等主流 App
const (
	DefaultPeriod = 30
	DefaultDigits = otp.DigitsSix
	DefaultAlgo   = otp.AlgorithmSHA1 // 没办法,兼容性
	SecretSize    = 32                // 32字节密钥
)

func generateSecret() []byte {
	b := make([]byte, SecretSize)
	_, _ = rand.Read(b)
	return b
}

func marshallSecret(secret []byte) string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
}

func unmarshallSecret(secret string) []byte {
	rawSecret, _ := base32.StdEncoding.
		WithPadding(base32.NoPadding).
		DecodeString(secret)
	return rawSecret
}

func CleanSecret(secret string) string {
	s := strings.ReplaceAll(secret, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ToUpper(s)
	return s
}

func FormatSecret(secret string) string {
	formatted := ""
	for i, c := range secret {
		if i%4 == 0 && i > 0 {
			formatted += "-"
		}
		formatted += string(c)
	}
	return formatted
}

func Verify(secret string, code string) (bool, error) {
	return totp.ValidateCustom(
		code,
		secret,
		time.Now().UTC(),
		totp.ValidateOpts{
			Period:    DefaultPeriod,
			Digits:    DefaultDigits,
			Algorithm: DefaultAlgo,
			Skew:      1, // 严格模式，不允许时间偏移（更安全）
		},
	)
}

func BuildURI(secret []byte, username, issuer string) string {
	key, _ := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: username,
		Secret:      secret,
		Algorithm:   DefaultAlgo,
		Period:      DefaultPeriod,
		Digits:      DefaultDigits,
	})
	return key.URL()
}
