package cli

import (
	"fmt"
	"io"
	"os"
	"pangolin/pkg/cli/models"
	"pangolin/pkg/utils"
	"strings"

	"github.com/spf13/cast"
)

func (c *jboxCli) GetDirectoryInfo(dirPath string,
	pagination *models.PaginationOption,
	order *models.OrderOption,
	filter string,
) (*models.DirectoryInfo, error) {
	if c.spaceInfo == nil {
		return nil, fmt.Errorf("获取目录'%s'信息失败，未登录！", dirPath)
	}

	url := fmt.Sprintf("/api/v1/directory/%s/%s/%s", c.spaceInfo.LibraryID, c.spaceInfo.SpaceID, dirPath)
	query := map[string]string{
		"access_token": c.spaceInfo.AccessToken,
	}

	if pagination != nil {
		query["page"] = cast.ToString(pagination.Page)
		query["page_size"] = cast.ToString(pagination.PageSize)
	}

	if order != nil {
		query["order_by"] = cast.ToString(order.By)
		query["order_by_type"] = cast.ToString(order.Type)
	}

	if len(filter) > 0 {
		query["filter"] = filter
	}

	resp, err := c.getRequest(c.baseUrl+url, query)
	if err != nil {
		return nil, err
	}

	if !utils.IsSuccessStatusCode(resp.StatusCode) {
		errMsg := models.ErrorMessage{}
		if unmarshalErr := utils.UnmarshalJson[models.ErrorMessage](resp, &errMsg); unmarshalErr == nil && errMsg.Message != "" {
			return nil, fmt.Errorf("服务器错误: %s", errMsg.Message)
		}
		return nil, fmt.Errorf("服务器响应状态码: %d", resp.StatusCode)
	}

	info := models.DirectoryInfo{}
	err = utils.UnmarshalJson[models.DirectoryInfo](resp, &info)
	return &info, err
}

func (c *jboxCli) List(path string) ([]FileEntry, error) {
	dirInfo, err := c.GetDirectoryInfo(path, nil, nil, "")
	if err != nil {
		return nil, err
	}

	ret := make([]FileEntry, 0, len(dirInfo.Contents))
	for _, content := range dirInfo.Contents {
		ret = append(ret, FileEntry{
			IsDir: content.IsDir(),
			Name:  content.Name,
		})
	}
	return ret, nil
}

func (c *jboxCli) GetFileDownloadInfo(filePath string) (*models.FileDownloadInfo, error) {
	if c.spaceInfo == nil {
		return nil, fmt.Errorf("获取文件'%s'下载信息失败，未登录！", filePath)
	}

	cleanPath := strings.TrimLeft(filePath, "/")
	url := fmt.Sprintf("/api/v1/file/%s/%s/%s", c.spaceInfo.LibraryID, c.spaceInfo.SpaceID, cleanPath)
	query := map[string]string{
		"info":         "",
		"access_token": c.spaceInfo.AccessToken,
	}

	resp, err := c.getRequest(c.baseUrl+url, query)
	if err != nil {
		return nil, err
	}

	if !utils.IsSuccessStatusCode(resp.StatusCode) {
		errMsg := models.ErrorMessage{}
		if unmarshalErr := utils.UnmarshalJson(resp, &errMsg); unmarshalErr == nil && errMsg.Message != "" {
			return nil, fmt.Errorf("服务器错误: %s", errMsg.Message)
		}
		return nil, fmt.Errorf("服务器响应状态码: %d", resp.StatusCode)
	}

	info := models.FileDownloadInfo{}
	err = utils.UnmarshalJson(resp, &info)
	return &info, err
}

func (c *jboxCli) DownloadFile(remotePath string, localPath string, onProgress models.DownloadProgressHandler) error {
	info, err := c.GetFileDownloadInfo(remotePath)
	if err != nil {
		return err
	}

	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	resp, err := utils.DoRequest("GET", info.CosUrl, c.headers, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	total := cast.ToInt64(info.Size)
	buf := make([]byte, 32*1024)
	var downloaded int64

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
			if onProgress != nil {
				onProgress(downloaded, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	return nil
}
