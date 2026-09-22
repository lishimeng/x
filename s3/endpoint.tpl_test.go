package s3

import "testing"

func TestEndpointTpl_Format(t *testing.T) {
	var tpl EndpointTpl = "xxx{region}xxxx"
	var r Region = "us-east-1"
	var s = tpl.Format(r)
	var expected = "xxxus-east-1xxxx"
	if s == expected {
		t.Log("ok")
	} else {
		t.Fatalf("fail, expected: %s, but: %s", expected, s)
	}
}
