package cmd

import (
	"fmt"
	"io"
	"pangolin/pkg/cli"
	"pangolin/pkg/cmd/models"
	"pangolin/pkg/path"
	stdpath "path"
	"sort"
	"strings"
)

type CdCommand struct {
	pathMgr path.PathManager
	jbox    cli.JboxClient
	args    []string
}

func NewCdCommand(pathMgr path.PathManager, jbox cli.JboxClient, args ...string) *CdCommand {
	return &CdCommand{
		pathMgr: pathMgr,
		jbox:    jbox,
		args:    args,
	}
}

func (c *CdCommand) Execute(in io.Reader, out io.Writer) error {
	if len(c.args) == 0 {
		return fmt.Errorf("请使用 help 查看指令使用方法")
	}
	target := "/"
	if len(c.args) > 0 {
		target = c.args[0]
	}

	dev, p := path.ParseDevicePath(target, path.CloudDisk)
	if dev != path.CloudDisk {
		return fmt.Errorf("cd 只支持 cloud 路径")
	}

	curr := c.pathMgr.CurrentPath().Path()
	var fullPath string
	if p == "" || p == "~" {
		fullPath = "/"
	} else if p[0] == '/' {
		fullPath = stdpath.Clean(p)
	} else {
		fullPath = stdpath.Join(curr, p)
	}

	_, err := c.jbox.List(fullPath)
	if err != nil {
		return fmt.Errorf("目录不存在: %s", target)
	}
	return c.pathMgr.ChangeDir(fullPath)
}

func (c *CdCommand) Name() string { return "cd" }
func (c *CdCommand) Help() string { return "Change directory (default: /)" }
func (c *CdCommand) Examples() string {
	return "cd\ncd /some/dir\ncd .."
}

func (c *CdCommand) Hint(args []string) []models.HintEntry {
	var lastArg string
	if len(args) > 0 {
		lastArg = args[len(args)-1]
	}

	if strings.HasPrefix(lastArg, "host:") {
		return nil
	}

	full := lastArg
	if full == "" || full[0] != '/' {
		full = stdpath.Join(c.pathMgr.CurrentPath().Path(), full)
	}

	if entries, err := c.jbox.List(full); err == nil {
		var dirs models.HintEntries
		for _, e := range entries {
			if e.IsDir {
				dirs = append(dirs, path.NewCloudDiskPath(stdpath.Join(full, e.Name)+"/", true))
			}
		}
		sort.Sort(dirs)
		return dirs
	}

	parent := stdpath.Dir(full)
	prefix := stdpath.Base(full)
	if parent == "." {
		parent = c.pathMgr.CurrentPath().Path()
	}

	entries, err := c.jbox.List(parent)
	if err != nil {
		return nil
	}

	var dirs models.HintEntries
	for _, e := range entries {
		if e.IsDir && strings.HasPrefix(e.Name, prefix) {
			dirs = append(dirs, path.NewCloudDiskPath(stdpath.Join(parent, e.Name)+"/", true))
		}
	}
	sort.Sort(dirs)
	return dirs
}
