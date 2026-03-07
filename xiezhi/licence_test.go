package xiezhi

import "testing"

func TestLicence_Marshall(t *testing.T) {

	signHandler, err := NewRsa([]byte(testPem), []byte(testKey))
	var lic Licence
	lic.Payload = []byte{0x01, 0x02, 0x04}
	sig, err := signHandler.Sign(lic.Payload)
	if err != nil {
		t.Fatal("signHandler.Sign:", err)
		return
	}
	lic.Signature = sig

	s := lic.Marshall()
	t.Log(s)

	var lic2 Licence
	err = lic2.Unmarshall(s) // license文件内容
	if err != nil {
		t.Fatalf("Unmarshall: %s", err)
		return
	}
	t.Log("marshall done")

	h, err := NewRsa([]byte(testPem))
	if err != nil {
		t.Fatalf("NewRsa: %s", err)
		return
	}
	err = lic2.Verify(h)
	if err != nil {
		t.Fatalf("Verify: %s", err)
		return
	}
	t.Log("verify done")
}

func TestSighWithObject(t *testing.T) {
	type SamplePayload struct {
		Payload
		A int64
		B string
		C float64
	}

	var payload SamplePayload

	h, err := NewRsa([]byte(testPem), []byte(testKey))
	if err != nil {
		t.Fatal("NewRsa:", err)
		return
	}
	var lic Licence
	err = lic.SignWithObject(h, payload)
	if err != nil {
		t.Fatal("SignWithObject:", err)
		return
	}
	s := lic.Marshall()
	t.Logf("signature: %s", s)
	t.Log("marshall done")
}

type SamplePayload struct {
	A int64
	B string
	C float64
}

func (payload *SamplePayload) Expired() error {
	return nil
}

func (payload *SamplePayload) SerialNumberCk() error {
	return nil
}

func (payload *SamplePayload) Verify() error {

	var err error
	err = payload.Expired()
	if err != nil {
		return err
	}
	err = payload.SerialNumberCk()
	if err != nil {
		return err
	}
	return nil
}

func TestSign(t *testing.T) {

	var payload = &SamplePayload{
		A: 1234,
		B: "hello world",
		C: 1.234,
	}

	h, err := NewRsa([]byte(testPem), []byte(testKey))
	if err != nil {
		t.Fatal("NewRsa:", err)
		return
	}

	content, err := Sign(payload, h)
	if err != nil {
		t.Fatal("Sign:", err)
		return
	}
	t.Log("sign done")

	var payload2 SamplePayload
	err = Verify(content, &payload2, h)
	if err != nil {
		t.Fatal("Verify:", err)
		return
	}
	t.Log("verify done")
}
