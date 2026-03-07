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
	var payload = make(map[string]any)
	payload["test"] = "test"
	payload["test2"] = "test2"
	payload["test3"] = "test3"

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
