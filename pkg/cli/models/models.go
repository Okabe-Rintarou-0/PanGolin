package models

import "strings"

const (
	OrderByAsc  = "asc"
	OrderByDesc = "desc"
)

type OrderOption struct {
	By   string
	Type string
}

type PaginationOption struct {
	Page     int
	PageSize int
}

type Session struct {
	JAAuthCookie  string `json:"JAAuthCookie"`
	UserToken     string `json:"userToken"`
	UserID        string `json:"userID"`
	Nickname      string `json:"nickname"`
	AccountUserID string `json:"accountUserId"`
}

type LoginPayload struct {
	Error   int64 `json:"error"`
	Payload struct {
		Sig string `json:"sig"`
		Ts  int64  `json:"ts"`
	} `json:"payload"`
	Type string `json:"type"`
}

type UserInfo struct {
	Entities Entities `json:"entities"`
	Errno    int64    `json:"errno"`
	Error    string   `json:"error"`
}

type Entities struct {
	AccountNo     string     `json:"accountNo"`
	Avatars       any        `json:"avatars"`
	Name          string     `json:"name"`
	UserType      string     `json:"userType"`
	UserStyleName string     `json:"userStyleName"`
	Email         string     `json:"email"`
	Code          string     `json:"code"`
	ExpireDate    string     `json:"expireDate"`
	Mobile        string     `json:"mobile"`
	Identities    []Identity `json:"identities"`
	OrganizeName  string     `json:"organizeName"`
	Status        string     `json:"status"`
	StatusEN      string     `json:"statusEN"`
	ResponseName  any        `json:"responseName"`
	OrganizeID    string     `json:"organizeId"`
	AuthAccounts  []any      `json:"authAccounts"`
}

type Identity struct {
	Kind            string    `json:"kind"`
	IsDefault       bool      `json:"isDefault"`
	DefaultOptional bool      `json:"defaultOptional"`
	Code            string    `json:"code"`
	UserType        string    `json:"userType"`
	Organize        Organize  `json:"organize"`
	TopOrganize     *Organize `json:"topOrganize"`
	Status          *string   `json:"status"`
	ExpireDate      *string   `json:"expireDate"`
	CreateDate      int64     `json:"createDate"`
	UpdateDate      int64     `json:"updateDate"`
	Gjm             *string   `json:"gjm"`
	FacultyType     any       `json:"facultyType"`
	PhotoURL        *string   `json:"photoUrl"`
	Type            *Organize `json:"type"`
	UserStyleName   string    `json:"userStyleName"`
}

type Organize struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type LoginResult struct {
	UserID        int64          `json:"userId"`
	UserToken     string         `json:"userToken"`
	ExpiresIn     int64          `json:"expiresIn"`
	Organizations []Organization `json:"organizations"`
	IsNewUser     bool           `json:"isNewUser"`
	Status        int            `json:"status"`
}

type Organization struct {
	ID             int64         `json:"id"`
	Name           string        `json:"name"`
	ExtensionData  ExtensionData `json:"extensionData"`
	LibraryID      string        `json:"libraryId"`
	LibraryCERT    string        `json:"libraryCert"`
	OrgUser        OrgUser       `json:"orgUser"`
	IsTemporary    bool          `json:"isTemporary"`
	IsLastSignedIn bool          `json:"isLastSignedIn"`
	Expired        bool          `json:"expired"`
	IPLimitEnabled bool          `json:"ipLimitEnabled"`
}

type ExtensionData struct {
	EnableDocPreview             bool               `json:"enableDocPreview"`
	EnableDocEdit                bool               `json:"enableDocEdit"`
	EnableMediaProcessing        bool               `json:"enableMediaProcessing"`
	Logo                         string             `json:"logo"`
	SsoWay                       string             `json:"ssoWay"`
	IPLimit                      IPLimit            `json:"ipLimit"`
	SyncWay                      string             `json:"syncWay"`
	UserLimit                    int64              `json:"userLimit"`
	ExpireTime                   string             `json:"expireTime"`
	EnableShare                  bool               `json:"enableShare"`
	LibraryFlag                  int64              `json:"libraryFlag"`
	AllowProduct                 string             `json:"allowProduct"`
	EditionConfig                EditionConfig      `json:"editionConfig"`
	EnableYufuLogin              bool               `json:"enableYufuLogin"`
	ShowOrgNameInUI              bool               `json:"showOrgNameInUI"`
	WatermarkOptions             WatermarkOptions   `json:"watermarkOptions"`
	EnableWeworkLogin            bool               `json:"enableWeworkLogin"`
	DefaultTeamOptions           DefaultTeamOptions `json:"defaultTeamOptions"`
	DefaultUserOptions           DefaultUserOptions `json:"defaultUserOptions"`
	AllowChangeNickname          bool               `json:"allowChangeNickname"`
	EnableOpenLDAPLogin          bool               `json:"enableOpenLdapLogin"`
	CacheDocPreviewTypes         string             `json:"cacheDocPreviewTypes"`
	EnableViewAllOrgUser         bool               `json:"enableViewAllOrgUser"`
	EnableWindowsAdLogin         bool               `json:"enableWindowsAdLogin"`
	OfficialDocPreviewTypes      string             `json:"officialDocPreviewTypes"`
	IsAccountNotDependentOnPhone bool               `json:"isAccountNotDependentOnPhone"`
}

type DefaultTeamOptions struct {
	DefaultRoleID  int64 `json:"defaultRoleId"`
	SpaceQuotaSize any   `json:"spaceQuotaSize"`
}

type DefaultUserOptions struct {
	Enabled                bool   `json:"enabled"`
	AllowPersonalSpace     bool   `json:"allowPersonalSpace"`
	PersonalSpaceQuotaSize string `json:"personalSpaceQuotaSize"`
}

type EditionConfig struct {
	EditionFlag               string `json:"editionFlag"`
	EnableOverseasPhoneNumber bool   `json:"enableOverseasPhoneNumber"`
	EnableOnlineEdit          bool   `json:"enableOnlineEdit"`
}

type IPLimit struct {
	LimitAdmin bool `json:"limitAdmin"`
}

type WatermarkOptions struct {
	ShareWatermarkType      int64 `json:"shareWatermarkType"`
	EnableShareWatermark    bool  `json:"enableShareWatermark"`
	PreviewWatermarkType    int64 `json:"previewWatermarkType"`
	DownloadWatermarkType   int64 `json:"downloadWatermarkType"`
	EnablePreviewWatermark  bool  `json:"enablePreviewWatermark"`
	EnableDownloadWatermark bool  `json:"enableDownloadWatermark"`
}

type OrgUser struct {
	Nickname           string `json:"nickname"`
	Role               string `json:"role"`
	Avatar             string `json:"avatar"`
	Deregister         bool   `json:"deregister"`
	Enabled            bool   `json:"enabled"`
	NeedChangePassword bool   `json:"needChangePassword"`
}

type DirectoryInfo struct {
	Path          []string        `json:"path"`
	SubDirCount   int64           `json:"subDirCount"`
	FileCount     int64           `json:"fileCount"`
	TotalNum      int64           `json:"totalNum"`
	ETag          string          `json:"eTag"`
	Contents      []*FileInfo     `json:"contents"`
	LocalSync     any             `json:"localSync"`
	AuthorityList map[string]bool `json:"authorityList"`
}

type MetaData map[string]any

type FileInfo struct {
	Path                     []string        `json:"path"`
	Name                     string          `json:"name"`
	Type                     string          `json:"type"`
	UserID                   string          `json:"userId"`
	CreationTime             string          `json:"creationTime"`
	ModificationTime         string          `json:"modificationTime"`
	VersionID                any             `json:"versionId"`
	VirusAuditStatus         int64           `json:"virusAuditStatus"`
	SensitiveWordAuditStatus int64           `json:"sensitiveWordAuditStatus"`
	ContentType              string          `json:"contentType"`
	Size                     string          `json:"size"`
	ETag                     string          `json:"eTag"`
	Crc64                    string          `json:"crc64"`
	MetaData                 MetaData        `json:"metaData"`
	AuthorityList            map[string]bool `json:"authorityList"`
	FileType                 string          `json:"fileType"`
	PreviewByDoc             bool            `json:"previewByDoc"`
	PreviewByCI              bool            `json:"previewByCI"`
	PreviewAsIcon            bool            `json:"previewAsIcon"`
}

func (d *DirectoryInfo) FullPath() string {
	return "/" + strings.Join(d.Path, "/")
}

func (f *FileInfo) FullPath() string {
	return "/" + strings.Join(f.Path, "/")
}

func (f *FileInfo) IsDir() bool {
	return f.Type == "dir"
}

type PersonalSpaceInfo struct {
	LibraryID   string `json:"libraryId"`
	SpaceID     string `json:"spaceId"`
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"`
	Status      int64  `json:"status"`
	Message     string `json:"message"`
}

type FileDownloadInfo struct {
	Type             string   `json:"type"`
	CreationTime     string   `json:"creationTime"`
	ModificationTime string   `json:"modificationTime"`
	ContentType      string   `json:"contentType"`
	Size             string   `json:"size"`
	ETag             string   `json:"eTag"`
	Crc64            string   `json:"crc64"`
	CosUrl           string   `json:"cosUrl"`
	AvailableCosUrls []string `json:"availableCosUrls"`
}

type DownloadProgressHandler = func(downloaded int64, total int64)

type UploadProgressHandler = func(uploaded int64, total int64)

type StartChunkUploadResult struct {
	ConfirmKey string                          `json:"confirmKey"`
	Domain     string                          `json:"domain"`
	Path       string                          `json:"path"`
	UploadID   string                          `json:"uploadId"`
	Parts      map[string]StartChunkUploadPart `json:"parts"`
	Expiration string                          `json:"expiration"`
}

type StartChunkUploadPart struct {
	Headers UploadPartHeaders `json:"headers"`
}

type UploadPartHeaders struct {
	XAmzDate          string `json:"x-amz-date"`
	XAmzContentSha256 string `json:"x-amz-content-sha256"`
	Authorization     string `json:"authorization"`
}

type ConfirmChunkUploadResult struct {
	Path                     []string `json:"path"`
	Name                     string   `json:"name"`
	Type                     string   `json:"type"`
	CreationTime             string   `json:"creationTime"`
	ModificationTime         string   `json:"modificationTime"`
	ContentType              string   `json:"contentType"`
	Size                     string   `json:"size"`
	ETag                     string   `json:"eTag"`
	Crc64                    string   `json:"crc64"`
	MetaData                 MetaData `json:"metaData"`
	IsOverwrittened          bool     `json:"isOverwrittened"`
	VirusAuditStatus         int64    `json:"virusAuditStatus"`
	SensitiveWordAuditStatus int64    `json:"sensitiveWordAuditStatus"`
	PreviewByDoc             bool     `json:"previewByDoc"`
	PreviewByCI              bool     `json:"previewByCI"`
	PreviewAsIcon            bool     `json:"previewAsIcon"`
	FileType                 string   `json:"fileType"`
}

type ChunkUploadInfo struct {
	Confirmed                  bool              `json:"confirmed"`
	Path                       []string          `json:"path"`
	Type                       string            `json:"type"`
	CreationTime               string            `json:"creationTime"`
	ConflictResolutionStrategy string            `json:"conflictResolutionStrategy"`
	Force                      bool              `json:"force"`
	UploadID                   string            `json:"uploadId"`
	Parts                      []ChunkUploadPart `json:"parts"`
}

type ChunkUploadPart struct {
	PartNumber   int64  `json:"PartNumber"`
	Size         int64  `json:"Size"`
	ETag         string `json:"ETag"`
	LastModified string `json:"LastModified"`
}
