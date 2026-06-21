package path

import (
	"fmt"
	"path/filepath"
	"strings"

	"pangolin/pkg/cmd/models"

	"github.com/charmbracelet/lipgloss"
)

var (
	devicePrefixes = []string{"cloud:", "host:"}
	dirStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#6EE7B7"))
)

type DeviceType = string

const (
	CloudDisk DeviceType = "cloud"
	Host      DeviceType = "host"
)

type CloudDiskPath struct {
	path  string
	isDir bool
}

func NewCloudDiskPath(path string, isDir bool) *CloudDiskPath {
	return &CloudDiskPath{
		path:  path,
		isDir: isDir,
	}
}

func (p *CloudDiskPath) Path() string {
	return p.path
}

func (p *CloudDiskPath) Compare(other models.HintEntry) int {
	o, ok := other.(*CloudDiskPath)
	if !ok {
		return strings.Compare(p.RealValue(), other.RealValue())
	}
	if p.isDir != o.isDir {
		if p.isDir {
			return -1
		}
		return 1
	}
	return strings.Compare(p.RealValue(), other.RealValue())
}

func (p *CloudDiskPath) DisplayValue() string {
	name := filepath.Base(p.path)
	if p.isDir {
		return dirStyle.Render(name + "/")
	}
	return name
}

func (p *CloudDiskPath) RealValue() string {
	return p.path
}

type Path struct {
	device string
	path   string
	isDir  bool
}

func NewPath(device string, path string, isDir bool) *Path {
	return &Path{
		device: device,
		path:   path,
		isDir:  isDir,
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

func (p *Path) Compare(other models.HintEntry) int {
	o, ok := other.(*Path)
	if !ok {
		return strings.Compare(p.RealValue(), other.RealValue())
	}
	if p.isDir != o.isDir {
		if p.isDir {
			return -1
		}
		return 1
	}
	return strings.Compare(p.RealValue(), other.RealValue())
}

func (p *Path) FullPath() string {
	return fmt.Sprintf("%s:%s", p.device, p.path)
}

func (p *Path) DisplayValue() string {
	name := filepath.Base(p.path)
	if p.isDir {
		return dirStyle.Render(name + "/")
	}
	return name
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
