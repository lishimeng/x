package s3

import "strings"

// EndpointTpl 模板, xxx{region}xxx格式
type EndpointTpl string

func (et EndpointTpl) Format(r Region) (s string) {
	s = strings.ReplaceAll(string(et), "{region}", string(r))
	return
}
