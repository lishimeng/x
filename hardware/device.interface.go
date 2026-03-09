package hardware

type OptFunc func(opt *Opt)

var opt *Opt

// WithNameFile 自定义设备名文件
var WithNameFile = func(name string) OptFunc {
	return func(opt *Opt) {
		opt.DeviceNamePath = name
	}
}

// WithSnFile 自定义序列号文件
var WithSnFile = func(name string) OptFunc {
	return func(opt *Opt) {
		opt.SerialNoPath = name
	}
}

func init() {
	opt = &Opt{}
}

func Setup(opts ...OptFunc) (err error) {
	for _, optFunc := range opts {
		optFunc(opt)
	}
	if len(opt.DeviceNamePath) == 0 {
		opt.DeviceNamePath = defaultDeviceNamePath
	}
	if len(opt.SerialNoPath) == 0 {
		opt.SerialNoPath = defaultSerialNumberPath
	}
	opt.Load()                 // 加载name
	opt.serialNumber = getSn() // 加载sn
	return
}
func GetSN() string {
	return opt.serialNumber
}

func GetName() string {
	return opt.deviceName
}
