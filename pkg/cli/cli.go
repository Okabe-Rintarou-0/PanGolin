package cli

import (
	"net/http"
	"pangolin/pkg/cli/models"
)

type FileEntry struct {
	IsDir bool
	Name  string
}

type JboxClient interface {
	Login(onQRReady func(string)) error
	HasSession() bool
 	SessionInfo() []string
	List(path string) ([]FileEntry, error)
	GetFileDownloadInfo(filePath string) (*models.FileDownloadInfo, error)
	DownloadFile(remotePath string, localPath string, onProgress models.DownloadProgressHandler) error
}

type jboxCli struct {
	cli         *http.Client
	sessionPath string
	session     *models.Session
	baseUrl     string
	headers     map[string]string
	spaceInfo   *models.PersonalSpaceInfo
}
