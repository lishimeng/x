package xiezhi

import "testing"

func TestGenDigest(t *testing.T) {
	var err error
	h, err := NewRsa([]byte(testPem), []byte(testKey))
	if err != nil {
		t.Fatal(err)
		return
	}
	var payload = []byte("这是明文")
	digest, err := h.Sign(payload)
	if err != nil {
		t.Fatal(err)
		return
	}
	var lic = Licence{
		Signature: digest,
		Payload:   payload,
	}
	s := lic.Marshall()
	t.Logf("license: %s\n", s)

	err = h.Verify(lic.Payload, lic.Signature)
	if err != nil {
		t.Fatal(err)
		return
	}

	t.Log("done")
}
