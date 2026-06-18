package cli

import (
	"fmt"
	"pangolin/pkg/cli/models"
	"pangolin/utils"

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
