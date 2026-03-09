package hardware

const (
	defaultSerialNumberPath = `/var/tmp/edge.code`
	defaultDeviceNamePath   = `/var/tmp/edge.name`
)

type Opt struct {
	DeviceNamePath string // 设备名文件, 可加载
	SerialNoPath   string // 序列号文件, 单向写入不可加载

	deviceName   string // 设备名
	serialNumber string // 序列号
}

// Save 保存数据
func (opt *Opt) Save() (err error) {

	// 保存序列号
	err = saveFile(opt.serialNumber, opt.SerialNoPath)
	if err != nil {
		return
	}
	// 保存device_name
	err = saveFile(opt.deviceName, opt.DeviceNamePath)
	if err != nil {
		return
	}
	return
}

func (opt *Opt) Load() {
	var err error
	// 加载device_name文件, 不报错
	data, err := loadFile(opt.DeviceNamePath)
	if err != nil {

	}
	name := isValidText(data)
	opt.deviceName = name
	// 不加载sn文件
	return
}
