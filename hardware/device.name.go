package hardware

import (
	"errors"
)

var errInvalidName = errors.New("网关名无效")

func UpdateName(name string) (err error) {
	name = isValidText(name)
	if name != "" {
		opt.deviceName = name
		err = saveName()
		return
	} else {
		return errInvalidName
	}
}

func saveName() (err error) {
	err = opt.Save()
	return
}
