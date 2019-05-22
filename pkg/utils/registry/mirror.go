package registry

import (
	"fmt"
)

const (
	DefaultRegistryMirror = "registry.cn-beijing.aliyuncs.com/yunionio"
)

func MirrorImage(name string, tag string, prefix string) string {
	if tag == "" {
		tag = "latest"
	}
	name = fmt.Sprintf("%s:%s", name, tag)
	if prefix != "" {
		name = fmt.Sprintf("%s-%s", prefix, name)
	}
	return fmt.Sprintf("%s/%s", DefaultRegistryMirror, name)
}
