package path_test

import (
	"strings"
	"testing"

	"pangolin/pkg/path"
)

func TestCloudDiskPath_DisplayValue(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		isDir bool
		want  string
	}{
		{
			name:  "directory without space",
			path:  "/互联网应用开发技术/",
			isDir: true,
			want:  "互联网应用开发技术/",
		},
		{
			name:  "directory with space",
			path:  "/互联网应用开发技术course files/",
			isDir: true,
			want:  "互联网应用开发技术course files/",
		},
		{
			name:  "file without space",
			path:  "/file.txt",
			isDir: false,
			want:  "file.txt",
		},
		{
			name:  "file with Chinese and space",
			path:  "/新建 Word 文档.docx",
			isDir: false,
			want:  "新建 Word 文档.docx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := path.NewCloudDiskPath(tt.path, tt.isDir)
			got := p.DisplayValue()
			if got != tt.want {
				t.Errorf("DisplayValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCloudDiskPath_RealValue(t *testing.T) {
	p := path.NewCloudDiskPath("/some/path/", true)
	if got := p.RealValue(); got != "/some/path/" {
		t.Errorf("RealValue() = %q, want %q", got, "/some/path/")
	}
}

func TestCloudDiskPath_Compare(t *testing.T) {
	tests := []struct {
		name string
		a, b *path.CloudDiskPath
		want int // <0 means a before b
	}{
		{
			name: "dir before file",
			a:    path.NewCloudDiskPath("/a/", true),
			b:    path.NewCloudDiskPath("/b.txt", false),
			want: -1,
		},
		{
			name: "file after dir",
			a:    path.NewCloudDiskPath("/a.txt", false),
			b:    path.NewCloudDiskPath("/b/", true),
			want: 1,
		},
		{
			name: "two dirs alphabetical",
			a:    path.NewCloudDiskPath("/a/", true),
			b:    path.NewCloudDiskPath("/b/", true),
			want: -1,
		},
		{
			name: "two files alphabetical",
			a:    path.NewCloudDiskPath("/a.txt", false),
			b:    path.NewCloudDiskPath("/b.txt", false),
			want: -1,
		},
		{
			name: "Chinese dirs alphabetical",
			a:    path.NewCloudDiskPath("/互联网应用开发技术/", true),
			b:    path.NewCloudDiskPath("/算法设计与分析/", true),
			want: strings.Compare("互联网应用开发技术/", "算法设计与分析/"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Compare(tt.b)
			if (got < 0) != (tt.want < 0) {
				t.Errorf("Compare(%q, %q) = %d, want sign %d", tt.a.RealValue(), tt.b.RealValue(), got, tt.want)
			}
		})
	}
}

func TestPath_DisplayValue(t *testing.T) {
	tests := []struct {
		name   string
		device string
		p      string
		isDir  bool
		want   string
	}{
		{
			name:   "cloud dir",
			device: "cloud",
			p:      "/dir/",
			isDir:  true,
			want:   "dir/",
		},
		{
			name:   "host file",
			device: "host",
			p:      "file.txt",
			isDir:  false,
			want:   "file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := path.NewPath(tt.device, tt.p, tt.isDir)
			got := p.DisplayValue()
			if got != tt.want {
				t.Errorf("DisplayValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPath_RealValue(t *testing.T) {
	p := path.NewPath("cloud", "/some/path/", true)
	want := "cloud:/some/path/"
	if got := p.RealValue(); got != want {
		t.Errorf("RealValue() = %q, want %q", got, want)
	}
}

func TestPath_Compare(t *testing.T) {
	tests := []struct {
		name string
		a, b *path.Path
		want int
	}{
		{
			name: "dir before file",
			a:    path.NewPath("cloud", "/a/", true),
			b:    path.NewPath("cloud", "/b.txt", false),
			want: -1,
		},
		{
			name: "file after dir",
			a:    path.NewPath("host", "a.txt", false),
			b:    path.NewPath("host", "b/", true),
			want: 1,
		},
		{
			name: "two dirs alphabetical",
			a:    path.NewPath("cloud", "/a/", true),
			b:    path.NewPath("cloud", "/b/", true),
			want: -1,
		},
		{
			name: "two files alphabetical",
			a:    path.NewPath("cloud", "/a.txt", false),
			b:    path.NewPath("cloud", "/b.txt", false),
			want: -1,
		},
		{
			name: "different device compares by RealValue",
			a:    path.NewPath("cloud", "/a/", true),
			b:    path.NewPath("host", "b/", true),
			want: strings.Compare("cloud:/a/", "host:b/"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Compare(tt.b)
			if (got < 0) != (tt.want < 0) {
				t.Errorf("Compare(%q, %q) = %d, want sign %d", tt.a.RealValue(), tt.b.RealValue(), got, tt.want)
			}
		})
	}
}
