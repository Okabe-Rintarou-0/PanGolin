package cli

import (
	"bytes"
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

func (c *jboxCli) CreateDirectory(dirPath string) error {
	if c.spaceInfo == nil {
		return fmt.Errorf("创建目录'%s'失败，未登录！", dirPath)
	}

	cleanPath := strings.TrimLeft(dirPath, "/")
	url := c.baseUrl + fmt.Sprintf("/api/v1/directory/%s/%s/%s", c.spaceInfo.LibraryID, c.spaceInfo.SpaceID, cleanPath)

	resp, err := c.putRequest(url, nil, map[string]string{
		"conflict_resolution_strategy": "ask",
		"access_token":                 c.spaceInfo.AccessToken,
	}, nil)
	if err != nil {
		return err
	}

	if !utils.IsSuccessStatusCode(resp.StatusCode) {
		errMsg := models.ErrorMessage{}
		if unmarshalErr := utils.UnmarshalJson[models.ErrorMessage](resp, &errMsg); unmarshalErr == nil && errMsg.Message != "" {
			return fmt.Errorf("创建目录失败: %s", errMsg.Message)
		}
		return fmt.Errorf("创建目录失败，服务器响应状态码: %d", resp.StatusCode)
	}

	errMsg := models.ErrorMessage{}
	err = utils.UnmarshalJson[models.ErrorMessage](resp, &errMsg)
	if err != nil {
		return err
	}
	if errMsg.Status != 0 {
		return fmt.Errorf("创建目录失败: %s", errMsg.Message)
	}
	return nil
}

func (c *jboxCli) StartChunkUpload(path string, chunkCount int64) (*models.StartChunkUploadResult, error) {
	if c.spaceInfo == nil {
		return nil, fmt.Errorf("开始上传'%s'失败，未登录！", path)
	}

	cleanPath := strings.TrimLeft(path, "/")
	url := c.baseUrl + fmt.Sprintf("/api/v1/file/%s/%s/%s", c.spaceInfo.LibraryID, c.spaceInfo.SpaceID, cleanPath)
	if chunkCount > 50 {
		chunkCount = 50
	}
	chunks := make([]int64, chunkCount)
	var i int64
	for i = 1; i <= chunkCount; i++ {
		chunks[i-1] = i
	}
	data := map[string]interface{}{}
	data["partNumberRange"] = chunks

	resp, err := c.postJson(url, map[string]string{
		"multipart":                    "null",
		"conflict_resolution_strategy": "overwrite",
		"access_token":                 c.spaceInfo.AccessToken,
	}, data)
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

	info := models.StartChunkUploadResult{}
	err = utils.UnmarshalJson[models.StartChunkUploadResult](resp, &info)
	return &info, err
}

func (c *jboxCli) Upload(ctx *models.StartChunkUploadResult, data []byte, partNumber int64, onProgress models.UploadProgressHandler) error {
	url := fmt.Sprintf("https://%s%s", ctx.Domain, ctx.Path)
	headerInfo := ctx.Parts[cast.ToString(partNumber)].Headers
	bufferSize := 81920 / 2
	reader := utils.NewProgressReader(bytes.NewReader(data), int64(len(data)), onProgress)
	body := &bufferedReadCloser{r: reader, bufSize: bufferSize}
	_, err := c.putRequest(url,
		map[string]string{
			"Accept":               "*/*",
			"Accept-Encoding":      "gzip, deflate, br",
			"Accept-Language":      "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7",
			"x-amz-date":           headerInfo.XAmzDate,
			"authorization":        headerInfo.Authorization,
			"x-amz-content-sha256": headerInfo.XAmzContentSha256,
		},
		map[string]string{
			"uploadId":   cast.ToString(ctx.UploadID),
			"partNumber": cast.ToString(partNumber),
		}, body)
	return err
}

func (c *jboxCli) ConfirmChunkUpload(confirmKey string) (*models.ConfirmChunkUploadResult, error) {
	if c.spaceInfo == nil {
		return nil, fmt.Errorf("确认上传失败，未登录！")
	}

	url := c.baseUrl + fmt.Sprintf("/api/v1/file/%s/%s/%s", c.spaceInfo.LibraryID, c.spaceInfo.SpaceID, confirmKey)

	resp, err := c.postRequest(url, map[string]string{
		"confirm":                      "null",
		"conflict_resolution_strategy": "overwrite",
		"access_token":                 c.spaceInfo.AccessToken,
	}, nil)
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

	info := models.ConfirmChunkUploadResult{}
	err = utils.UnmarshalJson(resp, &info)
	return &info, err
}

func (c *jboxCli) UploadFile(localPath string, remotePath string, onProgress models.UploadProgressHandler) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	fileSize := fi.Size()
	const minChunkSize = 5 * 1024 * 1024
	chunkCount := fileSize / minChunkSize
	if fileSize%minChunkSize != 0 {
		chunkCount++
	}
	if chunkCount > 50 {
		chunkCount = 50
	}
	if chunkCount == 0 {
		chunkCount = 1
	}
	chunkSize := fileSize / chunkCount
	if fileSize%chunkCount != 0 {
		chunkSize++
	}

	ctx, err := c.StartChunkUpload(remotePath, chunkCount)
	if err != nil {
		return err
	}

	buf := make([]byte, chunkSize)
	var uploaded int64
	for partNumber := int64(1); partNumber <= chunkCount; partNumber++ {
		n, readErr := f.Read(buf)
		if n > 0 {
			chunkData := make([]byte, n)
			copy(chunkData, buf[:n])

			partOnProgress := func(partUploaded, _ int64) {
				if onProgress != nil {
					onProgress(uploaded+partUploaded, fileSize)
				}
			}

			err = c.Upload(ctx, chunkData, partNumber, partOnProgress)
			if err != nil {
				return fmt.Errorf("上传分块 %d 失败: %w", partNumber, err)
			}
			uploaded += int64(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("读取文件分块 %d 失败: %w", partNumber, readErr)
		}
	}

	_, err = c.ConfirmChunkUpload(ctx.ConfirmKey)
	return err
}

type bufferedReadCloser struct {
	r       io.Reader
	bufSize int
}

func (b *bufferedReadCloser) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

func (b *bufferedReadCloser) Close() error {
	return nil
}
