package parser

import (
	"pangolin/pkg/cli"
	"pangolin/pkg/cmd"
	"pangolin/pkg/path"
	"strings"
)

// Parser 定义了解析终端输入的接口
type Parser interface {
	Parse(input string) cmd.Command
}

// PipeParser 是 Parser 接口的一个经典实现（Implementation）
// 它专门负责解析带有 "|" 管道符的命令链
type PipeParser struct {
	pathMgr path.PathManager
	jboxCli cli.JboxClient
}

// NewPipeParser 实例化一个管道解析器
func NewPipeParser(pathMgr path.PathManager, jboxCli cli.JboxClient) Parser {
	return &PipeParser{pathMgr, jboxCli}
}

// Parse 实现了 Parser 接口，解析原始字符串并组装成 Command
func (p *PipeParser) Parse(input string) cmd.Command {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	// 1. 按管道符 "|" 分割
	parts := strings.Split(input, "|")
	var cmds []cmd.Command

	for _, part := range parts {
		// 自动过滤并提取命令和参数
		args := strings.Fields(strings.TrimSpace(part))
		if len(args) == 0 {
			continue
		}

		cmdName := args[0]
		cmdArgs := args[1:]

		// 2. 核心路由映射
		switch cmdName {
		case "help":
			cmds = append(cmds, cmd.NewHelpCommand())
		case "ls":
			cmds = append(cmds, cmd.NewLsCommand(p.pathMgr, p.jboxCli, cmdArgs...))
		case "cd":
			cmds = append(cmds, cmd.NewCdCommand(p.pathMgr, p.jboxCli, cmdArgs...))
		case "cp":
			cmds = append(cmds, cmd.NewCpCommand(p.pathMgr, p.jboxCli, nil, cmdArgs...))
		default:
		}
	}

	if len(cmds) == 0 {
		return nil
	}

	// 3. 组装并返回管道命令
	return cmd.NewPipeline(cmds...)
}
