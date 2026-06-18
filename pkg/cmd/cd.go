package cmd

import (
	"io"
 	"pangolin/pkg/cli"
	"pangolin/pkg/path"
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
	target := "/"
	if len(c.args) > 0 {
		target = c.args[0]
	}
	return c.pathMgr.ChangeDir(target)
}
 
 func (c *CdCommand) Hint(input string) []string {
 	partial := ""
 	if idx := strings.Index(input, " "); idx >= 0 {
 		partial = strings.TrimSpace(input[idx+1:])
 	}
 
 	entries, err := c.jbox.List(c.pathMgr.CurrentPath().Path())
 	if err != nil {
 		return nil
 	}
 
 	var dirs []string
 	for _, e := range entries {
 		if e.IsDir && (partial == "" || strings.HasPrefix(e.Name, partial)) {
 			dirs = append(dirs, e.Name)
 		}
 	}
 	return dirs
 }
