package totp

type Secret struct {
	Secret         string `json:"secret"`
	Uri            string `json:"uri"`
	SecretFormated string `json:"secretFormated"`
}

type Opts struct {
	Issuer string
}
type BuildFunc func(opts *Opts)

var WithIssuer = func(issuer string) BuildFunc {
	return func(opts *Opts) {
		opts.Issuer = issuer
	}
}

type Handler struct {
	opts *Opts
}

func New(builder ...BuildFunc) (handler *Handler) {
	handler = new(Handler)
	handler.opts = &Opts{Issuer: "demo"}
	for _, b := range builder {
		b(handler.opts)
	}
	return
}

func (h *Handler) Generate(accountName string) (secret Secret) {
	bs := generateSecret()
	secret.Secret = marshallSecret(bs)
	secret.SecretFormated = FormatSecret(secret.Secret)
	secret.Uri = BuildURI(bs, accountName, h.opts.Issuer)
	return
}

func (h *Handler) Verify(secret string, code string) (ok bool, err error) {
	secret = CleanSecret(secret)
	return Verify(secret, code)
}
