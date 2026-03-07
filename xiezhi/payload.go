package xiezhi

type Payload interface {
	Verify() error
	Expired() error
	SerialNumberCk() error
}
