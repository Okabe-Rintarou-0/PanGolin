package cmd

import (
	"errors"
	"sort"
	"testing"

	"pangolin/pkg/cli"
	climodels "pangolin/pkg/cli/models"
	"pangolin/pkg/cmd/models"
)

type mockCli struct {
	listFn func(string) ([]cli.FileEntry, error)
}

func (m *mockCli) Login(_ func(string)) error                                  { return nil }
func (m *mockCli) HasSession() bool                                            { return true }
func (m *mockCli) SessionInfo() []string                                       { return nil }
func (m *mockCli) List(path string) ([]cli.FileEntry, error)                  { return m.listFn(path) }
func (m *mockCli) GetFileDownloadInfo(_ string) (*climodels.FileDownloadInfo, error) { return nil, nil }
func (m *mockCli) DownloadFile(_, _ string, _ climodels.DownloadProgressHandler) error { return nil }
func (m *mockCli) UploadFile(_, _ string, _ climodels.UploadProgressHandler) error    { return nil }
func (m *mockCli) CopyFile(_, _ string) error                                         { return nil }
func (m *mockCli) CopyDirectory(_, _ string) error                                    { return nil }
func (m *mockCli) CreateDirectory(_ string) error                                     { return nil }

func rootListing() []cli.FileEntry {
	return []cli.FileEntry{
		{Name: "算法设计与分析", IsDir: true},
		{Name: "互联网应用开发技术", IsDir: true},
		{Name: "互联网应用开发技术course files", IsDir: true},
		{Name: "新建 Word 文档.docx", IsDir: false},
		{Name: "file.txt", IsDir: false},
	}
}

func dirListing() []cli.FileEntry {
	return []cli.FileEntry{
		{Name: "lectures", IsDir: true},
		{Name: "hand1.pdf", IsDir: false},
		{Name: "新建 文档.pptx", IsDir: false},
	}
}

func TestHintCloudPath_Empty(t *testing.T) {
	mock := &mockCli{listFn: func(path string) ([]cli.FileEntry, error) {
		if path == "/" {
			return rootListing(), nil
		}
		return nil, errors.New("not found")
	}}

	hints := hintCloudPath("", "/", mock)
	if len(hints) == 0 {
		t.Fatal("expected hints for root listing")
	}

	dirDone := false
	for _, h := range hints {
		val := h.RealValue()
		isDir := len(val) > 0 && val[len(val)-1] == '/'
		if isDir && dirDone {
			t.Error("directory appears after file in sorted hints")
		}
		if !isDir {
			dirDone = true
		}
	}
}

func TestHintCloudPath_PrefixMatch(t *testing.T) {
	mock := &mockCli{listFn: func(path string) ([]cli.FileEntry, error) {
		if path == "/" {
			return rootListing(), nil
		}
		return nil, errors.New("not found")
	}}

	hints := hintCloudPath("互联", "/", mock)
	if len(hints) == 0 {
		t.Fatal("expected hints for prefix '互联'")
	}

	matched := 0
	for _, h := range hints {
		val := h.RealValue()
		if val == "cloud:/互联网应用开发技术" || val == "cloud:/互联网应用开发技术course files" {
			matched++
		}
	}
	if matched != 2 {
		t.Errorf("expected 2 matching dirs, got %d", matched)
	}
}

func TestHintCloudPath_ChineseFileWithSpace(t *testing.T) {
	mock := &mockCli{listFn: func(path string) ([]cli.FileEntry, error) {
		if path == "/" {
			return rootListing(), nil
		}
		return nil, errors.New("not found")
	}}

	// List root should include file with Chinese and space
	hints := hintCloudPath("", "/", mock)
	found := false
	for _, h := range hints {
		if h.RealValue() == "cloud:/新建 Word 文档.docx" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'cloud:/新建 Word 文档.docx' in root listing hints")
	}
}

func TestHintCloudPath_DirTraversal(t *testing.T) {
	mock := &mockCli{listFn: func(path string) ([]cli.FileEntry, error) {
		if path == "/算法设计与分析" {
			return dirListing(), nil
		}
		return nil, errors.New("not found")
	}}

	hints := hintCloudPath("算法设计与分析", "/", mock)
	if len(hints) == 0 {
		t.Fatal("expected hints for dir traversal")
	}

	found := map[string]bool{
		"cloud:/算法设计与分析/lectures":  false,
		"cloud:/算法设计与分析/hand1.pdf": false,
	}
	for _, h := range hints {
		val := h.RealValue()
		if _, ok := found[val]; ok {
			found[val] = true
		}
	}
	for k, v := range found {
		if !v {
			t.Errorf("missing hint: %s", k)
		}
	}
}

func TestCpHint_AfterTabCompleteThenSpace(t *testing.T) {
	mock := &mockCli{listFn: func(path string) ([]cli.FileEntry, error) {
		return nil, errors.New("not found")
	}}
	cmd := NewCpCommand(nil, nil, mock, nil, "cloud:/src/", "")

	hints := cmd.Hint([]string{"cloud:/src/", ""})
	if len(hints) == 0 {
		t.Fatal("expected hint for empty dest")
	}
	// Current logic: empty lastArg with no colon → hintDevice("") matches cloud first
	if hints[0].RealValue() != "cloud:" {
		t.Errorf("expected cloud: prefix hint, got %q", hints[0].RealValue())
	}
}

func TestHintCloudPath_NoMatch(t *testing.T) {
	mock := &mockCli{listFn: func(path string) ([]cli.FileEntry, error) {
		if path == "/" {
			return rootListing(), nil
		}
		return nil, errors.New("not found")
	}}

	hints := hintCloudPath("zzznonexistent", "/", mock)
	if len(hints) != 0 {
		t.Errorf("expected no hints for non-matching prefix, got %d", len(hints))
	}
}

func TestHintCloudPath_HostPrefixReturnsNil(t *testing.T) {
	mock := &mockCli{}
	hints := hintCloudPath("host:something", "/", mock)
	if hints != nil {
		t.Error("expected nil hints for host: prefix in cloud path")
	}
}

func TestHintCloudPath_RelativePath(t *testing.T) {
	mock := &mockCli{listFn: func(path string) ([]cli.FileEntry, error) {
		if path == "/互联网应用开发技术" {
			return dirListing(), nil
		}
		return nil, errors.New("not found")
	}}

	hints := hintCloudPath("互联网应用开发技术", "/", mock)
	if len(hints) == 0 {
		t.Fatal("expected hints for relative path")
	}
}

func TestHintCloudPath_SortOrder(t *testing.T) {
	mock := &mockCli{listFn: func(path string) ([]cli.FileEntry, error) {
		if path == "/" {
			return rootListing(), nil
		}
		return nil, errors.New("not found")
	}}

	hints := hintCloudPath("", "/", mock)
	if len(hints) < 3 {
		t.Fatal("need at least 3 hints to test sort order")
	}

	if !sort.IsSorted(models.HintEntries(hints)) {
		t.Error("hints not sorted (dirs first, then alphabetical)")
	}
}

func TestHintCloudPath_CurrPathSubdir(t *testing.T) {
	mock := &mockCli{listFn: func(path string) ([]cli.FileEntry, error) {
		if path == "/算法设计与分析" {
			return dirListing(), nil
		}
		return nil, errors.New("not found")
	}}

	hints := hintCloudPath("", "/算法设计与分析", mock)
	if len(hints) == 0 {
		t.Fatal("expected hints for subdirectory")
	}

	found := map[string]bool{
		"cloud:/算法设计与分析/lectures":  false,
		"cloud:/算法设计与分析/hand1.pdf": false,
	}
	for _, h := range hints {
		val := h.RealValue()
		if _, ok := found[val]; ok {
			found[val] = true
		}
	}
	for k, v := range found {
		if !v {
			t.Errorf("missing hint: %s", k)
		}
	}
}
