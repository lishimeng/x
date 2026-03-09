package hardware

import "testing"

func TestGenId(t *testing.T) {
	var err error
	err = Setup(WithNameFile("./name.test.txt"), WithSnFile("./sn.test.txt"))
	if err != nil {
		t.Fatalf("Setup err: %v", err)
		return
	}
	err = UpdateName("thisisname")
	if err != nil {
		t.Fatalf("Update err: %v", err)
		return
	}
	t.Logf("sn:%s", GetSN())
	t.Logf("device_name:%s", GetName())
}
