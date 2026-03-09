package hardware

import (
	"crypto/hmac"
	"crypto/md5"
	"fmt"

	"github.com/denisbrodbeck/machineid"
)

// SN生成算法

// 操作系统编号, 装系统时随机生成
var osId string

func init() {
	osId, _ = machineid.ID()
	mac := hmac.New(md5.New, []byte(osId+"lism"))
	mac.Write([]byte(osId))
	bs := mac.Sum(nil)
	osId = fmt.Sprintf("%02x", bs)
}

func getSn() string {
	return osId
}
