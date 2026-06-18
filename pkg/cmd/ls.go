package cmd

import (
 	"fmt"
 	"io"
	"pangolin/pkg/cli"
 	"sort"
)

type LsCommand struct {
	cli      cli.JboxClient
	currPath string
	args     []string
}

func NewLsCommand(cli cli.JboxClient, currPath string, args ...string) *LsCommand {
	return &LsCommand{
		cli:      cli,
		currPath: currPath,
		args:     args,
	}
}

func (l *LsCommand) Execute(in io.Reader, out io.Writer) error {
	var dirPath string
	if len(l.args) == 0 {
		dirPath = l.currPath
	} else {
		dirPath = l.args[0]
	}
	entries, err := l.cli.List(dirPath)
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
 
 	const (
 		green = "\033[32m"
 		reset = "\033[0m"
 	)
 
 	for i, entry := range entries {
 		if i != 0 {
 			if _, err := fmt.Fprint(out, "  "); err != nil {
 				return err
 			}
 		}
 
		if entry.IsDir {
 			if _, err := fmt.Fprintf(out, "%s%s%s", green, entry.Name, reset); err != nil {
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
