package cmd

import (
	"fmt"
	"io"
	"os"
	"pangolin/pkg/cli"
	"pangolin/pkg/cmd/models"
	"pangolin/pkg/path"
	"pangolin/pkg/tui/handle"
	stdpath "path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxConcurrency = 5
	maxRetries     = 3
)

type CpCommand struct {
	cli        cli.JboxClient
	pathMgr    path.PathManager
	srcPath    string
	destPath   string
	recursive  bool
	pbarHandle *handle.ProgressBarHandle
	onProgress func(current, total int)
}

func NewCpCommand(pbarHandle *handle.ProgressBarHandle, pathMgr path.PathManager, cli cli.JboxClient, onProgress func(int, int), args ...string) *CpCommand {
	recursive := false
	remaining := args
	if len(remaining) > 0 && remaining[0] == "-r" {
		recursive = true
		remaining = remaining[1:]
	}
	src := ""
	dest := ""
	if len(remaining) > 0 {
		src = remaining[0]
	}
	if len(remaining) > 1 {
		dest = remaining[1]
	}
	return &CpCommand{
		cli:        cli,
		pathMgr:    pathMgr,
		srcPath:    src,
		destPath:   dest,
		recursive:  recursive,
		onProgress: onProgress,
		pbarHandle: pbarHandle,
	}
}

func (c *CpCommand) Execute(in io.Reader, out io.Writer) error {
	if c.srcPath == "" {
		return fmt.Errorf("用法: cp [-r] <src> [dst]")
	}

	srcDevice, src := path.ParseDevicePath(c.srcPath, path.CloudDisk)
	destDefault := path.Host
	if srcDevice == path.Host {
		destDefault = path.CloudDisk
	}
	destDevice, dest := path.ParseDevicePath(c.destPath, destDefault)

	switch {
	case srcDevice == path.CloudDisk && destDevice == path.Host:
		return c.executeDownload(src, dest, out)
	case srcDevice == path.Host && destDevice == path.CloudDisk:
		return c.executeUpload(src, dest, out)
	case srcDevice == path.CloudDisk && destDevice == path.CloudDisk:
		return c.executeCloudCopy(src, dest, out)
	case srcDevice == path.Host && destDevice == path.Host:
		return c.executeHostCopy(src, dest, out)
	default:
		return fmt.Errorf("不支持从 %s 复制到 %s", srcDevice, destDevice)
	}
}

func (c *CpCommand) executeDownload(src, dest string, out io.Writer) error {
	if !strings.HasPrefix(src, "/") {
		src = stdpath.Join(c.pathMgr.CurrentPath().Path(), src)
	}

	if dest == "" {
		dest = filepath.Base(src)
	} else {
		info, err := os.Stat(dest)
		if err == nil && info.IsDir() {
			dest = filepath.Join(dest, filepath.Base(src))
		}
	}

	if entries, listErr := c.cli.List(src); listErr == nil {
		if !c.recursive {
			return fmt.Errorf("%s 是一个目录，请使用 cp -r 来复制目录", filepath.Base(src))
		}
		return c.copyDirBFS(src, dest, entries, out)
	}

	if c.recursive {
		return fmt.Errorf("%s 不是一个目录", filepath.Base(src))
	}
	return c.copyFile(src, dest, out)
}

func (c *CpCommand) executeUpload(src, dest string, out io.Writer) error {
	if !strings.HasPrefix(dest, "/") {
		dest = stdpath.Join(c.pathMgr.CurrentPath().Path(), dest)
	}

	if dest == "" || strings.HasSuffix(dest, "/") {
		dest = stdpath.Join(dest, filepath.Base(src))
	} else if entries, err := c.cli.List(dest); err == nil {
		_ = entries
		dest = stdpath.Join(dest, filepath.Base(src))
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("无法访问本地路径'%s': %w", src, err)
	}

	if srcInfo.IsDir() {
		if !c.recursive {
			return fmt.Errorf("%s 是一个目录，请使用 cp -r 来复制目录", src)
		}
		return c.uploadDirBFS(src, dest, out)
	}
	if c.recursive {
		return fmt.Errorf("%s 不是一个目录", src)
	}
	return c.uploadFile(src, dest, out)
}

func isSubDir(parent, child string) bool {
	if parent == child {
		return true
	}
	parent = strings.TrimRight(parent, "/") + "/"
	return strings.HasPrefix(child, parent)
}

func (c *CpCommand) executeCloudCopy(src, dest string, out io.Writer) error {
	if !strings.HasPrefix(src, "/") {
		src = stdpath.Join(c.pathMgr.CurrentPath().Path(), src)
	}
	if !strings.HasPrefix(dest, "/") {
		dest = stdpath.Join(c.pathMgr.CurrentPath().Path(), dest)
	}

	if dest == "" || strings.HasSuffix(dest, "/") {
		dest = stdpath.Join(dest, stdpath.Base(src))
	} else if entries, err := c.cli.List(dest); err == nil {
		_ = entries
		if stdpath.Base(src) != stdpath.Base(dest) {
			dest = stdpath.Join(dest, stdpath.Base(src))
		}
	}

	if isSubDir(src, dest) {
		return fmt.Errorf("不能将目录复制到自身内部")
	}

	if entries, listErr := c.cli.List(src); listErr == nil {
		if !c.recursive {
			return fmt.Errorf("%s 是一个目录，请使用 cp -r 来复制目录", stdpath.Base(src))
		}
		return c.cloudCopyDirBFS(src, dest, entries, out)
	}

	if c.recursive {
		return fmt.Errorf("%s 不是一个目录", stdpath.Base(src))
	}
	return c.cloudCopyFile(src, dest, out)
}

func (c *CpCommand) executeHostCopy(src, dest string, out io.Writer) error {
	// Resolve relative paths for dest only — src is used as-is
	if dest == "" || strings.HasSuffix(dest, "/") || strings.HasSuffix(dest, "\\") {
		dest = filepath.Join(dest, filepath.Base(src))
	} else if info, err := os.Stat(dest); err == nil && info.IsDir() {
		dest = filepath.Join(dest, filepath.Base(src))
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("无法访问本地路径'%s': %w", src, err)
	}

	if srcInfo.IsDir() {
		if !c.recursive {
			return fmt.Errorf("%s 是一个目录，请使用 cp -r 来复制目录", src)
		}
		return c.hostCopyDirBFS(src, dest, out)
	}
	if c.recursive {
		return fmt.Errorf("%s 不是一个目录", src)
	}
	return c.hostCopyFile(src, dest, out)
}

func (c *CpCommand) copyFile(src, dest string, out io.Writer) error {
	if c.pbarHandle != nil {
		c.pbarHandle.Create()
	}
	err := c.cli.DownloadFile(src, dest, func(downloaded, total int64) {
		if c.pbarHandle != nil {
			c.pbarHandle.Set(int(downloaded), int(total))
		}
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s -> %s\n", src, dest)
	return nil
}

type dirEntry struct {
	src  string
	dest string
}

type fileTask struct {
	src  string
	dest string
}

func (c *CpCommand) copyDirBFS(rootSrc, rootDest string, rootEntries []cli.FileEntry, out io.Writer) error {
	os.MkdirAll(rootDest, 0755)

	var files []fileTask
	queue := []dirEntry{{rootSrc, rootDest}}
	entryMap := map[string][]cli.FileEntry{rootSrc: rootEntries}
	visited := map[string]bool{rootSrc: true}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		entries, ok := entryMap[item.src]
		if !ok {
			var err error
			entries, err = c.cli.List(item.src)
			if err != nil {
				return err
			}
		}

		for _, e := range entries {
			subSrc := stdpath.Join(item.src, e.Name)
			subDest := filepath.Join(item.dest, e.Name)
			if e.IsDir {
				os.MkdirAll(subDest, 0755)
				if !visited[subSrc] {
					visited[subSrc] = true
					queue = append(queue, dirEntry{subSrc, subDest})
				}
			} else {
				files = append(files, fileTask{subSrc, subDest})
			}
		}
	}

	if len(files) == 0 {
		return nil
	}
	return c.downloadFiles(files, out)
}

func (c *CpCommand) uploadFile(src, dest string, out io.Writer) error {
	if c.pbarHandle != nil {
		c.pbarHandle.Create()
	}
	err := c.cli.UploadFile(src, dest, func(uploaded, total int64) {
		if c.pbarHandle != nil {
			c.pbarHandle.Set(int(uploaded), int(total))
		}
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s -> %s\n", src, dest)
	return nil
}

type uploadTask struct {
	src  string
	dest string
}

func (c *CpCommand) uploadDirBFS(rootSrc, rootDest string, out io.Writer) error {
	c.cli.CreateDirectory(rootDest)

	var files []uploadTask
	queue := []uploadTask{{rootSrc, rootDest}}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		entries, err := os.ReadDir(item.src)
		if err != nil {
			return fmt.Errorf("读取本地目录'%s'失败: %w", item.src, err)
		}

		for _, e := range entries {
			subSrc := filepath.Join(item.src, e.Name())
			subDest := stdpath.Join(item.dest, e.Name())
			if e.IsDir() {
				c.cli.CreateDirectory(subDest)
				queue = append(queue, uploadTask{subSrc, subDest})
			} else {
				files = append(files, uploadTask{subSrc, subDest})
			}
		}
	}

	if len(files) == 0 {
		return nil
	}
	return c.uploadFiles(files, out)
}

func (c *CpCommand) uploadFiles(files []uploadTask, _ io.Writer) error {
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrency)

	var mu sync.Mutex
	completed := 0
	total := len(files)

	var once sync.Once
	var firstErr error

	if c.pbarHandle != nil {
		c.pbarHandle.Create()
	}
	for _, f := range files {
		wg.Add(1)
		go func(f uploadTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			err := retry(maxRetries, func() error {
				return c.cli.UploadFile(f.src, f.dest, nil)
			})
			if err != nil {
				once.Do(func() { firstErr = fmt.Errorf("%s 上传失败: %w", f.src, err) })
			}

			mu.Lock()
			completed++
			if c.onProgress != nil {
				c.onProgress(completed, total)
			}
			if c.pbarHandle != nil {
				c.pbarHandle.Set(completed, total)
			}
			mu.Unlock()
		}(f)
	}

	wg.Wait()
	if firstErr != nil && c.pbarHandle != nil {
		c.pbarHandle.SetError(firstErr)
	}
	return firstErr
}

func (c *CpCommand) cloudCopyFile(src, dest string, out io.Writer) error {
	err := c.cli.CopyFile(src, dest)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s -> %s\n", src, dest)
	return nil
}

func (c *CpCommand) cloudCopyDirBFS(rootSrc, rootDest string, rootEntries []cli.FileEntry, out io.Writer) error {
	// Use server-side async directory copy API
	err := c.cli.CopyDirectory(rootSrc, rootDest)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s -> %s\n", rootSrc, rootDest)
	return nil
}

func (c *CpCommand) hostCopyFile(src, dest string, out io.Writer) error {
	err := c.copyLocalFile(src, dest)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s -> %s\n", src, dest)
	return nil
}

func (c *CpCommand) copyLocalFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	fi, err := os.Stat(src)
	if err == nil {
		os.Chmod(dest, fi.Mode())
	}
	return nil
}

func (c *CpCommand) hostCopyDirBFS(rootSrc, rootDest string, out io.Writer) error {
	os.MkdirAll(rootDest, 0755)

	var files []uploadTask
	queue := []uploadTask{{rootSrc, rootDest}}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		entries, err := os.ReadDir(item.src)
		if err != nil {
			return err
		}

		for _, e := range entries {
			subSrc := filepath.Join(item.src, e.Name())
			subDest := filepath.Join(item.dest, e.Name())
			if e.IsDir() {
				os.MkdirAll(subDest, 0755)
				queue = append(queue, uploadTask{subSrc, subDest})
			} else {
				files = append(files, uploadTask{subSrc, subDest})
			}
		}
	}

	if len(files) == 0 {
		return nil
	}
	return c.hostCopyFiles(files, out)
}

func (c *CpCommand) hostCopyFiles(files []uploadTask, _ io.Writer) error {
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrency)

	var mu sync.Mutex
	completed := 0
	total := len(files)

	var once sync.Once
	var firstErr error

	if c.pbarHandle != nil {
		c.pbarHandle.Create()
	}
	for _, f := range files {
		wg.Add(1)
		go func(f uploadTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			err := retry(maxRetries, func() error {
				return c.copyLocalFile(f.src, f.dest)
			})
			if err != nil {
				once.Do(func() { firstErr = fmt.Errorf("%s 复制失败: %w", f.src, err) })
			}

			mu.Lock()
			completed++
			if c.onProgress != nil {
				c.onProgress(completed, total)
			}
			if c.pbarHandle != nil {
				c.pbarHandle.Set(completed, total)
			}
			mu.Unlock()
		}(f)
	}

	wg.Wait()
	if firstErr != nil && c.pbarHandle != nil {
		c.pbarHandle.SetError(firstErr)
	}
	return firstErr
}

func (c *CpCommand) Name() string { return "cp" }
func (c *CpCommand) Help() string { return "Copy file/dir between cloud and local host" }
func (c *CpCommand) Examples() string {
	return "cp file.txt ~/Desktop/\ncp host:file.txt cloud:dir/\ncp -r mydir ./backup/\ncp -r host:mydir cloud:mydir/\ncp cloud:file.txt cloud:backup/file.txt\ncp host:file.txt host:backup/"
}

func (c *CpCommand) downloadFiles(files []fileTask, _ io.Writer) error {
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrency)

	var mu sync.Mutex
	completed := 0
	total := len(files)

	var once sync.Once
	var firstErr error

	if c.pbarHandle != nil {
		c.pbarHandle.Create()
	}
	for _, f := range files {
		wg.Add(1)
		go func(f fileTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			err := retry(maxRetries, func() error {
				return c.cli.DownloadFile(f.src, f.dest, nil)
			})
			if err != nil {
				once.Do(func() { firstErr = fmt.Errorf("%s 下载失败: %w", f.src, err) })
			}

			mu.Lock()
			completed++
			if c.onProgress != nil {
				c.onProgress(completed, total)
			}
			if c.pbarHandle != nil {
				c.pbarHandle.Set(completed, total)
			}
			mu.Unlock()
		}(f)
	}

	wg.Wait()
	if firstErr != nil && c.pbarHandle != nil {
		c.pbarHandle.SetError(firstErr)
	}
	return firstErr
}

func retry(attempts int, fn func() error) error {
	var err error
	for i := range attempts {
		err = fn()
		if err == nil {
			return nil
		}
		if i < attempts-1 {
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
		}
	}
	return err
}

func (c *CpCommand) Hint(args []string) []models.HintEntry {
	var lastArg string
	if len(args) > 0 {
		lastArg = args[len(args)-1]
	}

	device, p, found := strings.Cut(lastArg, ":")
	if !found {
		if deviceType, ok := c.hintDevice(device); ok {
			return []models.HintEntry{path.NewPath(deviceType, "", false)}
		}
	} else {
		if device == path.CloudDisk {
			return hintCloudPath(p, c.pathMgr.CurrentPath().Path(), c.cli)
		}
		if device == path.Host {
			return hintLocalPath(p)
		}
	}
	return nil
}

func (c *CpCommand) hintDevice(device string) (path.DeviceType, bool) {
	if strings.HasPrefix(path.CloudDisk, device) {
		return path.CloudDisk, true
	}
	if strings.HasPrefix(path.Host, device) {
		return path.Host, true
	}
	return "", false
}

func hintCloudPath(partial, currPath string, cli cli.JboxClient) []models.HintEntry {
	if strings.HasPrefix(partial, "host:") {
		return nil
	}
	pp := strings.TrimPrefix(partial, "cloud:")
	full := pp
	if full == "" || full[0] != '/' {
		full = stdpath.Join(currPath, full)
	}

	if entries, err := cli.List(full); err == nil {
		var hints models.HintEntries
		for _, e := range entries {
			name := e.Name
			if e.IsDir {
				name += "/"
			}
			hints = append(hints, path.NewPath(path.CloudDisk, stdpath.Join(full, name), e.IsDir))
		}
		sort.Sort(hints)
		return hints
	}

	parent := stdpath.Dir(full)
	prefix := stdpath.Base(full)
	if parent == "." {
		parent = currPath
	}

	entries, err := cli.List(parent)
	if err != nil {
		return nil
	}

	var hints models.HintEntries
	for _, e := range entries {
		if strings.HasPrefix(e.Name, prefix) {
			name := e.Name
			if e.IsDir {
				name += "/"
			}
			hints = append(hints, path.NewPath(path.CloudDisk, stdpath.Join(parent, name), e.IsDir))
		}
	}
	sort.Sort(hints)
	return hints
}

func hintLocalPath(partial string) []models.HintEntry {
	return listLocalDir(partial)
}

func listLocalDir(partial string) []models.HintEntry {
	if partial == "" {
		entries, err := os.ReadDir(".")
		if err != nil {
			return nil
		}
		var hints models.HintEntries
		for _, e := range entries {
			name := e.Name()
			isdir := e.IsDir()
			if isdir {
				name += string(filepath.Separator)
			}
			hints = append(hints, path.NewPath(path.Host, name, isdir))
		}
		sort.Sort(hints)
		return hints
	}

	if strings.HasSuffix(partial, "/") || strings.HasSuffix(partial, "\\") {
		entries, err := os.ReadDir(partial)
		if err != nil {
			return nil
		}
		var hints models.HintEntries
		for _, e := range entries {
			name := partial + e.Name()
			isdir := e.IsDir()
			if isdir {
				name += string(filepath.Separator)
			}
			hints = append(hints, path.NewPath(path.Host, name, isdir))
		}
		sort.Sort(hints)
		return hints
	}

	parent := filepath.Dir(partial)
	filter := filepath.Base(partial)

	entries, err := os.ReadDir(parent)
	if err != nil {
		return nil
	}
	var hints models.HintEntries
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), filter) {
			continue
		}
		name := e.Name()
		if parent != "." {
			name = filepath.Join(parent, name)
		}
		isdir := e.IsDir()
		if isdir {
			name += string(filepath.Separator)
		}
		hints = append(hints, path.NewPath(path.Host, name, isdir))
	}
	sort.Sort(hints)
	return hints
}
