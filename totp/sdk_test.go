package totp

import "testing"

func TestName(t *testing.T) {
	h := New(WithIssuer("issuer.demo.io"))
	secret := h.Generate("me")
	t.Logf("secret: %s", secret.Secret)
	t.Logf("formated: %s", secret.SecretFormated)
	t.Logf("uri: %s", secret.Uri)
}

func TestVerify(t *testing.T) {
	var s = `3LEYXUNTDMDBVFSFZ7L5S43FUSNKQTSXDEHIZUTVZVXYDXUOEHBQ`
	var code = `302652` // 配合验证器
	h := New(WithIssuer("issuer.demo.io"))
	t.Logf("code: %s", code)
	t.Logf("secret: %s", s)
	ok, err := h.Verify(s, code)
	t.Log(ok, err)
}
