package cmd

import "io"

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

	// 如果只有一个命令，直接执行，不需要建立管道
	if len(p.commands) == 1 {
		return p.commands[0].Execute(in, out)
	}

	// 用于收集异步进程错误的 channel
	errChan := make(chan error, len(p.commands)-1)
	var pipesCloseFuncs []func() error

	// 驱动链条：当前的输入源
	currentIn := in

	// 遍历执行除了最后一个命令之外的所有中间命令
	for i := 0; i < len(p.commands)-1; i++ {
		pr, pw := io.Pipe()
		pipesCloseFuncs = append(pipesCloseFuncs, pw.Close)

		// 并发异步启动中间进程
		go func(cmd Command, childIn io.Reader, childOut *io.PipeWriter, index int) {
			err := cmd.Execute(childIn, childOut)
			_ = childOut.Close() // 当前进程结束后，必须关闭管道写端，通知下一级 EOF
			errChan <- err
		}(p.commands[i], currentIn, pw, i)

		// 下一个命令的输入源，变成当前管道的读取端
		currentIn = pr
	}

	// 在当前主线程阻塞执行最后一个命令，它直接输出到外部传入的 out
	lastErr := p.commands[len(p.commands)-1].Execute(currentIn, out)

	// 收集其他并发进程的错误
	for i := 0; i < len(p.commands)-1; i++ {
		if err := <-errChan; err != nil && lastErr == nil {
			lastErr = err
		}
	}

	return lastErr
}
 
 func (p *Pipeline) Hint(input string) []string {
 	if len(p.commands) == 0 {
 		return nil
 	}
 	return p.commands[len(p.commands)-1].Hint(input)
 }
