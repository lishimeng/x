# hardware(硬件信息)

计算设备序列号

```go
package sample
import "github.com/lishimeng/x/hardware"
import "testing"

// WithSnFile: 设置序列号文件
// WithNameFile: 设置设备名文件

func TestGenId(t *testing.T) {
	var err error
	err = hardware.Setup(
		hardware.WithNameFile("./name.test.txt"), 
		hardware.WithSnFile("./sn.test.txt"))
	if err != nil {
		t.Fatalf("Setup err: %v", err)
		return
	}
	err = hardware.UpdateName("thisisname")
	if err != nil {
		t.Fatalf("Update err: %v", err)
		return
	}
	t.Logf("sn:%s", hardware.GetSN())
	t.Logf("device_name:%s", hardware.GetName())
}

```