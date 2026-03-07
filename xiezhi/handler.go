package xiezhi

type Handler interface {
	Verify(payload []byte, signature []byte) error
	Sign(input []byte) (signature []byte, err error)
}
