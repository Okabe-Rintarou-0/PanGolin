package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"pangolin/pkg/cli"
	"pangolin/pkg/cmd/models"
	"pangolin/pkg/path"
	stdpath "path"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func init() {
	file, _ := os.Create("tmp.log")
	log.SetOutput(file)
}

type LsCommand struct {
	pathMgr path.PathManager
	jbox    cli.JboxClient
	args    []string
}

func NewLsCommand(pathMgr path.PathManager, jbox cli.JboxClient, args ...string) *LsCommand {
	return &LsCommand{
		pathMgr: pathMgr,
		jbox:    jbox,
		args:    args,
	}
}

func (l *LsCommand) Execute(in io.Reader, out io.Writer) error {
	var dirPath string
	if len(l.args) == 0 {
		dirPath = l.pathMgr.CurrentPath().Path()
	} else {
		dirPath = l.args[0]
		if dirPath == "." || dirPath == ".." {
			dirPath = stdpath.Join(l.pathMgr.CurrentPath().Path(), dirPath)
		}
	}
	entries, err := l.jbox.List(dirPath)
	if err != nil {
		return err
	}

	// Sort: directories first, then alphabetically by name
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})

	dirStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00875F"))

	for i, entry := range entries {
		if i != 0 {
			if _, err := fmt.Fprint(out, "  "); err != nil {
				return err
			}
		}

		if entry.IsDir {
			if _, err := fmt.Fprint(out, dirStyle.Render(entry.Name)); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprint(out, entry.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *LsCommand) Hint(args []string) []models.HintEntry {
	var lastArg string
	if len(args) > 0 {
		lastArg = args[len(args)-1]
	}

	if strings.HasPrefix(lastArg, "host:") {
		return nil
	}

	full := lastArg
	if full == "" || full[0] != '/' {
		full = stdpath.Join(l.pathMgr.CurrentPath().Path(), full)
	}
	if entries, err := l.jbox.List(full); err == nil {
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
		parent = l.pathMgr.CurrentPath().Path()
	}

	entries, err := l.jbox.List(parent)
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
func (l *LsCommand) Name() string { return "ls" }
func (l *LsCommand) Help() string { return "List files and directories" }
func (l *LsCommand) Examples() string {
	return "ls\nls /some/path"
}
