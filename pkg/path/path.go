package path

import "fmt"

type DeviceType = string

const (
	CloudDisk DeviceType = "cloud"
	Host      DeviceType = "host"
)

type Path struct {
	device string
	path   string
}

func NewPath(device string, path string) *Path {
	return &Path{
		device, path,
	}
}

func (p *Path) IsValid() bool {
	return p.device == CloudDisk || p.device == Host
}

func (p *Path) Device() DeviceType {
	return p.device
}

func (p *Path) Path() string {
	return p.path
}

func (p *Path) FullPath() string {
	return fmt.Sprintf("%s:%s", p.device, p.path)
}
