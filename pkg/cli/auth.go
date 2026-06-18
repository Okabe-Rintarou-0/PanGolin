package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"pangolin/pkg/cli/models"
	"pangolin/pkg/utils"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var (
	authUrl, _ = url.Parse("https://jaccount.sjtu.edu.cn/jaccount")
)

func NewJboxClient() JboxClient {
	configDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	cli := &http.Client{Jar: jar}
	sessionPath := filepath.Join(configDir, "pangolin", "session.json")
	return &jboxCli{
		cli:         cli,
		sessionPath: sessionPath,
		session:     &models.Session{},
		baseUrl:     "https://pan.sjtu.edu.cn",
		headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		},
	}
}

func (c *jboxCli) Login(onQRReady func(string)) error {
	session, _ := c.getPersistentSession()
	jaAuthCookie := ""
	if session != nil && len(session.JAAuthCookie) > 0 {
		jaAuthCookie = session.JAAuthCookie
	}

	if len(jaAuthCookie) > 0 {
		c.session.JAAuthCookie = jaAuthCookie
		err := c.loginInternal()
		if err == nil {
			c.saveSession()
			return nil
		}
		// session expired/invalid — fall through to QR login
	}

	err := c.qrcodeLogin(onQRReady)
	if err != nil {
		return err
	}

	c.saveSession()
	return nil
}

func (c *jboxCli) HasSession() bool {
	session, err := c.getPersistentSession()
	if err != nil {
		return false
	}
	return session != nil && len(session.JAAuthCookie) > 0
}

type jboxUserResponse struct {
	AccountUserID string `json:"accountUserId"`
	Nickname      string `json:"nickname"`
	Email         string `json:"email"`
	Enabled       bool   `json:"enabled"`
}

func (c *jboxCli) fetchUserInfo() error {
	path := fmt.Sprintf("/user/v1/user/1/%s", c.session.UserID)
	resp, err := c.getRequest(c.baseUrl+path, map[string]string{
		"user_token":           c.session.UserToken,
		"with_belonging_teams": "false",
		"pf":                   "",
	})
	if err != nil {
		return err
	}
	user := jboxUserResponse{}
	if err := utils.UnmarshalJson(resp, &user); err != nil {
		return err
	}
	c.session.Nickname = user.Nickname
	c.session.AccountUserID = user.AccountUserID
	return nil
}

func (c *jboxCli) saveSession() error {
	data, err := json.Marshal(c.session)
	if err != nil {
		return err
	}
	dir := filepath.Dir(c.sessionPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(c.sessionPath, data, 0600)
}

func (c *jboxCli) SessionInfo() []string {
	info := make([]string, 0, 4)
	if c.session.Nickname != "" {
		info = append(info, fmt.Sprintf("User: %s", c.session.Nickname))
	} else if c.session.UserID != "" {
		info = append(info, fmt.Sprintf("User: %s", c.session.UserID))
	}
	info = append(info, "Session: active")
	if c.spaceInfo != nil && c.spaceInfo.ExpiresIn > 0 {
		expires := time.Now().Add(time.Duration(c.spaceInfo.ExpiresIn) * time.Second)
		info = append(info, fmt.Sprintf("Expires: %s", expires.Format("15:04:05")))
	}
	if c.session.AccountUserID != "" {
		info = append(info, fmt.Sprintf("Account: %s", c.session.AccountUserID))
	}
	return info
}

func (c *jboxCli) getPersonalSpaceInfo() (*models.PersonalSpaceInfo, error) {
	url := "/user/v1/space/1/personal"
	resp, err := c.postRequest(c.baseUrl+url, map[string]string{
		"user_token": c.session.UserToken,
	}, nil)
	if err != nil {
		return nil, err
	}

	errMessage := models.ErrorMessage{}
	if !c.isSuccessStatusCode(resp.StatusCode) {
		err = utils.UnmarshalJson(resp, &errMessage)
		if err != nil {
			return nil, fmt.Errorf("获取个人信息失败！服务器响应%d", resp.StatusCode)
		}
		return nil, fmt.Errorf("%s", errMessage.Message)
	}
	info := models.PersonalSpaceInfo{}
	err = utils.UnmarshalJson(resp, &info)
	if err != nil {
		return nil, err
	}

	if info.Status != 0 {
		return nil, fmt.Errorf("获取个人信息失败！服务器返回失败：%s", info.Message)
	}
	return &info, nil
}

func (c *jboxCli) initSpace() error {
	var (
		info *models.PersonalSpaceInfo
		err  error
	)
	t := 3
	for t > 0 {
		t -= 1
		info, err = c.getPersonalSpaceInfo()
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}
	c.spaceInfo = info
	return nil
}

func (c *jboxCli) loginInternal() error {
	c.cli.Jar.SetCookies(authUrl, []*http.Cookie{
		{Name: "JAAuthCookie", Value: c.session.JAAuthCookie},
	})
	resp, err := c.cli.Get("https://pan.sjtu.edu.cn/user/v1/sign-in/sso-login-redirect/xpw8ou8y")
	if err != nil {
		return err
	}

	if !c.isSuccessStatusCode(resp.StatusCode) {
		return fmt.Errorf("登录网盘失败，服务器响应：%d", resp.StatusCode)
	}

	if strings.Contains(resp.Request.URL.Host, "jaccount") {
		return fmt.Errorf("登录网盘失败，未成功认证")
	}

	reg := regexp.MustCompile("code=(.+?)&state=")
	matches := reg.FindStringSubmatch(resp.Request.URL.String())

	if len(matches) == 0 {
		panic(fmt.Errorf("登录网盘失败，未找到回调code"))
	}
	code := matches[len(matches)-1]
	nextUrl := "https://pan.sjtu.edu.cn/user/v1/sign-in/verify-account-login/xpw8ou8y?device_id=Chrome+116.0.0.0&type=sso&credential=" + code

	resp, err = c.cli.Post(nextUrl, "", nil)
	if !c.isSuccessStatusCode(resp.StatusCode) {
		return fmt.Errorf("登录网盘失败，服务器响应：%d", resp.StatusCode)
	}

	loginRes := models.LoginResult{}
	err = utils.UnmarshalJson(resp, &loginRes)
	if err != nil {
		return err
	}

	if loginRes.Status != 0 {
		return fmt.Errorf("登录网盘失败，服务器响应：%d", loginRes.Status)
	}

	if len(loginRes.UserToken) != 128 {
		return fmt.Errorf("登录网盘失败，user token 无效")
	}

	c.session.UserToken = loginRes.UserToken
	c.session.UserID = strconv.FormatInt(loginRes.UserID, 10)
	err = c.initSpace()
	if err != nil {
		return err
	}
	_ = c.fetchUserInfo() // best-effort, won't block login
	return nil
}

func (c *jboxCli) getPersistentSession() (*models.Session, error) {
	var content []byte
	file, err := os.Open(c.sessionPath)
	if file != nil {
		content, err = io.ReadAll(file)
		if err != nil {
			return nil, err
		}
	}
	defer file.Close()

	session := &models.Session{}
	err = json.Unmarshal(content, session)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (c *jboxCli) isSuccessStatusCode(statusCode int) bool {
	return statusCode >= 200 && statusCode <= 299
}

func (c *jboxCli) checkUserInfo() error {
	resp, err := c.cli.Get("https://my.sjtu.edu.cn/api/resource/my/info")
	if err != nil {
		return err
	}
	if !c.isSuccessStatusCode(resp.StatusCode) {
		return fmt.Errorf("获取用户信息时出错，服务器响应验证码：%d", resp.StatusCode)
	}

	user := models.UserInfo{}
	err = utils.UnmarshalJson(resp, &user)

	if user.Errno != 0 {
		return fmt.Errorf("获取用户信息时出错，服务器返回错误：%s", user.Error)
	}

	return nil
}

func (c *jboxCli) getUuid() (string, error) {
	resp, err := c.cli.Get("https://my.sjtu.edu.cn/ui/appmyinfo")
	if err != nil {
		return "", err
	}
	redirect := strings.Contains(resp.Request.URL.String(), "https://jaccount.sjtu.edu.cn/jaccount/jalogin")
	if resp.StatusCode == http.StatusOK && !redirect {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取 uuid 失败，服务器返回状态%d", resp.StatusCode)
	}

	defer resp.Body.Close()
	var bytes []byte
	bytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	fmt.Println("here")
	pattern := regexp.MustCompile(
		`uuid\s*[:=]\s*["']?([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})["']?`,
	)
	result := pattern.FindAllStringSubmatch(string(bytes), -1)
	if result == nil || len(result[0]) < 2 {
		return "", fmt.Errorf("获取 uuid 失败，没有找到 uuid")
	}
	uuid := result[0][1]
	return uuid, nil
}

func initWebsocket(uuid string) (*websocket.Conn, error) {
	uri, _ := url.Parse(fmt.Sprintf("wss://jaccount.sjtu.edu.cn/jaccount/sub/%s", uuid))
	c, _, err := websocket.DefaultDialer.Dial(uri.String(), nil)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func getQRCodeURL(uuid, sig string, ts int64) string {
	return fmt.Sprintf("https://jaccount.sjtu.edu.cn/jaccount/confirmscancode?uuid=%s&ts=%d&sig=%s", uuid, ts, sig)
}

func sendUpdateQRCodeMessage(ws *websocket.Conn) {
	message := "{ \"type\": \"UPDATE_QR_CODE\" }"
	if err := ws.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
		_ = ws.Close()
	}
}

func sendUpdateQRCodeMessageWorker(ws *websocket.Conn, ctx context.Context) {
	sendUpdateQRCodeMessage(ws)
	ticker := time.Tick(time.Second * 50)
	for {
		select {
		case <-ticker:
			sendUpdateQRCodeMessage(ws)
		case <-ctx.Done():
			return
		}
	}
}

func (c *jboxCli) handleScanSuccess(uuid string) error {
	resp, err := c.cli.Get(fmt.Sprintf("https://jaccount.sjtu.edu.cn/jaccount/expresslogin?uuid=%s", uuid))
	if err != nil {
		return err
	}

	if !c.isSuccessStatusCode(resp.StatusCode) {
		return fmt.Errorf("expresslogin失败，服务器返回%d", resp.StatusCode)
	}

	redirect := strings.Contains(resp.Request.URL.String(), "https://jaccount.sjtu.edu.cn/jaccount/jalogin")
	if resp.StatusCode == http.StatusOK && redirect {
		return fmt.Errorf("expresslogin失败，未认证")
	}

	return nil
}

func (c *jboxCli) qrcodeLogin(onQRReady func(string)) error {
	var (
		err       error
		uuid      string
		ws        *websocket.Conn
		message   []byte
		payload   models.LoginPayload
		tp        string
		messageTp int
	)
	uuid, err = c.getUuid()
	if err != nil {
		return err
	}
	ws, err = initWebsocket(uuid)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	go sendUpdateQRCodeMessageWorker(ws, ctx)
	for {
		messageTp, message, err = ws.ReadMessage()
		if messageTp != websocket.TextMessage {
			continue
		}
		err = json.Unmarshal(message, &payload)
		if err != nil {
			return fmt.Errorf("消息格式错误：%s", err.Error())
		}
		if payload.Error != 0 {
			return fmt.Errorf("登录错误：%d", payload.Error)
		}
		tp = strings.ToUpper(payload.Type)
		if tp == "UPDATE_QR_CODE" {
			qrcodeURL := getQRCodeURL(uuid, payload.Payload.Sig, payload.Payload.Ts)
			if onQRReady != nil {
				onQRReady(qrcodeURL)
			}
		} else if tp == "LOGIN" {
			if err = c.handleScanSuccess(uuid); err != nil {
				return err
			}
			break
		}
	}

	cookies := c.cli.Jar.Cookies(authUrl)
	for _, cookie := range cookies {
		if cookie.Name == "JAAuthCookie" {
			c.session.JAAuthCookie = cookie.Value
			return c.loginInternal()
		}
	}
	return fmt.Errorf("未读取到 JAAuthCookie！")
}
