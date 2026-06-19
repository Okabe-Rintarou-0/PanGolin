package path

import (
	"fmt"
	"path/filepath"
	"strings"
)

var (
	devicePrefixes = []string{"cloud:", "host:"}
)

type DeviceType = string

const (
	CloudDisk DeviceType = "cloud"
	Host      DeviceType = "host"
)

type CloudDiskPath struct {
	path string
}

func NewCloudDiskPath(path string) *CloudDiskPath {
	return &CloudDiskPath{
		path,
	}
}

func (p *CloudDiskPath) Path() string {
	return p.path
}

func (p *CloudDiskPath) DisplayValue() string {
	return filepath.Base(p.path)
}

func (p *CloudDiskPath) RealValue() string {
	return p.path
}

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

func (p *Path) DisplayValue() string {
	return filepath.Base(p.path)
}
func (p *Path) RealValue() string {
	return p.FullPath()
}

func ParseDevicePath(s string, defaultDevice DeviceType) (DeviceType, string) {
	if after, ok := strings.CutPrefix(s, CloudDisk+":"); ok {
		p := after
		if p == "" {
			p = "/"
		}
		return CloudDisk, p
	}
	if after, ok := strings.CutPrefix(s, Host+":"); ok {
		p := after
		if p == "" {
			p = "."
		}
		return Host, p
	}
	return defaultDevice, s
}

func DevicePrefixes() []string {
	return []string{CloudDisk + ":", Host + ":"}
}
