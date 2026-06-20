package cmd

import (
	"io"
	"pangolin/pkg/cmd/models"
)

type Pipeline struct {
	commands []Command
}

func NewPipeline(cmds ...Command) *Pipeline {
	return &Pipeline{commands: cmds}
}

func (p *Pipeline) Execute(in io.Reader, out io.Writer) error {
	if len(p.commands) == 0 {
		return nil
	}

	if len(p.commands) == 1 {
		return p.commands[0].Execute(in, out)
	}

	// 错误收集 channel
	errChan := make(chan error, len(p.commands)-1)

	// 用于存储所有创建出来的管道，以便后续统一清理
	var pipeReaders []*io.PipeReader
	var pipeWriters []*io.PipeWriter

	// 确保无论发生什么，最后都关闭所有管道，释放被卡住的 Goroutine
	defer func() {
		for _, pw := range pipeWriters {
			_ = pw.Close()
		}
		for _, pr := range pipeReaders {
			_ = pr.Close()
		}
	}()

	currentIn := in

	// 遍历执行中间命令
	for i := 0; i < len(p.commands)-1; i++ {
		pr, pw := io.Pipe()
		pipeReaders = append(pipeReaders, pr)
		pipeWriters = append(pipeWriters, pw)

		// 异步启动中间进程
		go func(cmd Command, childIn io.Reader, childOut *io.PipeWriter) {
			err := cmd.Execute(childIn, childOut)
			// 每一个中间命令执行完，立刻关闭自己的写端，通知下一级 EOF
			_ = childOut.Close()
			errChan <- err
		}(p.commands[i], currentIn, pw)

		// 下一个命令的输入源
		currentIn = pr
	}

	// 在主线程阻塞执行最后一个命令
	lastErr := p.commands[len(p.commands)-1].Execute(currentIn, out)

	for i := 0; i < len(p.commands)-1; i++ {
		if err := <-errChan; err != nil && lastErr == nil {
			lastErr = err
		}
	}

	return lastErr
}

func (p *Pipeline) Commands() []Command {
	return p.commands
}

func (p *Pipeline) Name() string {
	if len(p.commands) > 0 {
		return p.commands[len(p.commands)-1].Name()
	}
	return "pipeline"
}

func (p *Pipeline) Help() string { return "" }

func (p *Pipeline) Examples() string { return "" }

func (p *Pipeline) Hint(args []string) []models.HintEntry {
	if len(p.commands) == 0 {
		return nil
	}
	return p.commands[len(p.commands)-1].Hint(args)
}
