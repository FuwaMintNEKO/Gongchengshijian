package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"school-trade/middleware"
	"school-trade/models"
	"school-trade/store"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// OAuthConfig OAuth2 配置（从环境变量读取，提供默认值）
type OAuthConfig struct {
	UASBaseURL     string // UAS 后端 API 地址（换取 token、userinfo）
	UASFrontendURL string // UAS 前端地址（授权页跳转），空则从请求推断
	ClientID       string // 应用 AppID
	ClientSecret   string // 应用 AppSecret
	RedirectURI    string // 回调地址，空则从请求推断
}

func loadOAuthConfig() OAuthConfig {
	cfg := OAuthConfig{
		UASBaseURL:     getEnv("UAS_BASE_URL", "http://localhost:8081"),
		UASFrontendURL: getEnv("UAS_FRONTEND_URL", ""),
		ClientID:       getEnv("UAS_CLIENT_ID", "KK790SCHOOLTRADE"),
		ClientSecret:   getEnv("UAS_CLIENT_SECRET", ""),
		RedirectURI:    getEnv("UAS_REDIRECT_URI", ""),
	}
	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getRequestScheme 从请求推断协议（支持反向代理）
func getRequestScheme(c *gin.Context) string {
	if c.Request.TLS != nil {
		return "https"
	}
	if xfp := c.GetHeader("X-Forwarded-Proto"); xfp != "" {
		return xfp
	}
	return "http"
}

// resolveRedirectURI 解析回调地址：优先用配置，否则从请求推断
// 推断规则：scheme://当前host/oauth/callback
func (h *OAuthHandler) resolveRedirectURI(c *gin.Context) string {
	if h.Cfg.RedirectURI != "" {
		return h.Cfg.RedirectURI
	}
	return fmt.Sprintf("%s://%s/oauth/callback", getRequestScheme(c), c.Request.Host)
}

// resolveUASFrontendURL 解析 UAS 前端地址：优先用配置，否则从请求推断
// 推断规则：与当前请求同 IP，端口取 UAS 前端默认端口 8082
func (h *OAuthHandler) resolveUASFrontendURL(c *gin.Context) string {
	if h.Cfg.UASFrontendURL != "" {
		return h.Cfg.UASFrontendURL
	}
	scheme := getRequestScheme(c)
	host := c.Request.Host
	// 提取 IP（去掉端口），如果原 host 没有端口则直接用
	ip, _, err := net.SplitHostPort(host)
	if err != nil {
		ip = host
	}
	return fmt.Sprintf("%s://%s:8082", scheme, ip)
}

// OAuthHandler 处理与UAS的OAuth2对接
type OAuthHandler struct {
	Store *store.DBStore
	Cfg   OAuthConfig
}

func NewOAuthHandler(s *store.DBStore) *OAuthHandler {
	return &OAuthHandler{
		Store: s,
		Cfg:   loadOAuthConfig(),
	}
}

// GetConfig 返回前端使用的OAuth配置（不暴露secret）
// GET /api/oauth/config
func (h *OAuthHandler) GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "ok",
		Data: gin.H{
			"uasBaseUrl":     h.Cfg.UASBaseURL,
			"uasFrontendUrl": h.resolveUASFrontendURL(c),
			"clientId":       h.Cfg.ClientID,
			"redirectUri":    h.resolveRedirectURI(c),
			"enabled":        h.Cfg.ClientSecret != "",
		},
	})
}

// Login 重定向到UAS授权页
// GET /api/oauth/login
func (h *OAuthHandler) Login(c *gin.Context) {
	// 前端地址用于授权后跳回原页面
	frontRedirect := c.Query("redirect")
	if frontRedirect == "" {
		frontRedirect = "/"
	}

	// state 防止CSRF，这里用 frontRedirect 编码后作为 state
	state := frontRedirect

	// 授权页地址与回调地址：优先用环境变量配置，未配置则从当前请求推断
	// 这样本地和云服务器（同 IP 不同端口）都无需手动配置即可工作
	uasFrontendURL := h.resolveUASFrontendURL(c)
	redirectURI := h.resolveRedirectURI(c)

	// UAS前端授权页地址（UAS前端会调用 /api/oauth/authorize 获取应用信息）
	uasAuthorizeURL := fmt.Sprintf("%s/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&state=%s&scope=userinfo",
		uasFrontendURL,
		url.QueryEscape(h.Cfg.ClientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(state),
	)
	c.Redirect(http.StatusFound, uasAuthorizeURL)
}

// Callback 接收UAS回调的授权码，换取token，获取用户信息，登录/注册本地用户
// GET /api/oauth/callback
func (h *OAuthHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	errParam := c.Query("error")
	state := c.Query("state")

	if errParam != "" {
		h.redirectToFront(c, "/", "授权失败: "+errParam)
		return
	}

	if code == "" {
		h.redirectToFront(c, "/", "未获取到授权码")
		return
	}

	if h.Cfg.ClientSecret == "" {
		h.redirectToFront(c, "/", "OAuth未配置ClientSecret")
		return
	}

	// 用 code 换取 access_token
	// redirect_uri 必须与 Login 时传给 UAS 的一致，否则 UAS 会拒绝
	redirectURI := h.resolveRedirectURI(c)
	tokenResp, err := h.exchangeCodeForToken(code, redirectURI)
	if err != nil {
		h.redirectToFront(c, "/", "换取令牌失败: "+err.Error())
		return
	}

	accessToken, _ := tokenResp["access_token"].(string)
	if accessToken == "" {
		h.redirectToFront(c, "/", "未获取到访问令牌")
		return
	}

	// 用 access_token 获取用户信息
	userInfo, err := h.fetchUserInfo(accessToken)
	if err != nil {
		h.redirectToFront(c, "/", "获取用户信息失败: "+err.Error())
		return
	}

	// 登录或注册本地用户
	localUser, err := h.loginOrCreateUser(userInfo)
	if err != nil {
		h.redirectToFront(c, "/", "登录失败: "+err.Error())
		return
	}

	// 生成本地JWT
	token, expiresAt, err := middleware.GenerateToken(localUser.ID, localUser.Username, localUser.Role)
	if err != nil {
		h.redirectToFront(c, "/", "生成令牌失败")
		return
	}

	// 设置 HttpOnly Cookie
	c.SetCookie("sso_token", token, int(time.Until(time.Unix(expiresAt, 0)).Seconds()), "/", "", false, true)

	// 跳转到登录页，由前端把 token 保存到 localStorage（前端通过 cookie fallback 也能用，
	// 但显式保存可确保 SPA 内部跳转不丢失登录态）
	// state 即为授权前用户所在页面，授权后跳回
	frontRedirect := state
	if frontRedirect == "" {
		frontRedirect = "../index.html"
	}
	// 走 login.html 中转：login.html 检测到 uas_token 参数后会保存并跳转
	loginURL := "/pages/login.html?uas_token=" + url.QueryEscape(token) + "&redirect=" + url.QueryEscape(frontRedirect)
	c.Redirect(http.StatusFound, loginURL)
}

// exchangeCodeForToken 用授权码向UAS换取access_token
func (h *OAuthHandler) exchangeCodeForToken(code, redirectURI string) (map[string]interface{}, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {h.Cfg.ClientID},
		"client_secret": {h.Cfg.ClientSecret},
		"redirect_uri":  {redirectURI},
	}

	resp, err := http.PostForm(h.Cfg.UASBaseURL+"/api/oauth/token", form)
	if err != nil {
		return nil, fmt.Errorf("请求UAS失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if errMsg, ok := result["error"].(string); ok {
		desc, _ := result["error_description"].(string)
		return nil, fmt.Errorf("%s: %s", errMsg, desc)
	}

	return result, nil
}

// fetchUserInfo 用access_token获取UAS用户信息
func (h *OAuthHandler) fetchUserInfo(accessToken string) (map[string]interface{}, error) {
	req, _ := http.NewRequest("GET", h.Cfg.UASBaseURL+"/api/oauth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求用户信息失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析用户信息失败: %w", err)
	}

	if errMsg, ok := result["error"].(string); ok {
		desc, _ := result["error_description"].(string)
		return nil, fmt.Errorf("%s: %s", errMsg, desc)
	}

	return result, nil
}

// loginOrCreateUser 根据UAS返回的用户信息登录或注册本地用户
func (h *OAuthHandler) loginOrCreateUser(uasUser map[string]interface{}) (*models.User, error) {
	db := h.Store.GetDB()
	if db == nil {
		return nil, fmt.Errorf("数据库未连接")
	}

	uasUserID := fmt.Sprintf("%v", uasUser["user_id"])
	userType, _ := uasUser["user_type"].(string)
	if userType == "" {
		userType = "personal"
	}

	// UAS用户唯一标识：类型_用户ID
	uasUniqueID := userType + "_" + uasUserID

	// 1. 先按 uas_user_id 查找已绑定的本地用户
	user, err := scanUserRow(db.QueryRow(
		"SELECT id, username, password, nickname, avatar, phone, email, role, created_at, updated_at FROM users WHERE uas_user_id = ?",
		uasUniqueID,
	))
	if err == nil {
		// 已绑定：更新昵称/头像/手机号（保持同步）
		nickname, _ := uasUser["nickname"].(string)
		realName, _ := uasUser["real_name"].(string)
		phone, _ := uasUser["phone"].(string)
		email, _ := uasUser["email"].(string)
		avatar, _ := uasUser["avatar"].(string)
		if nickname == "" {
			nickname = realName
		}
		_, _ = db.Exec(
			"UPDATE users SET nickname=?, phone=?, email=?, avatar=?, updated_at=? WHERE id=?",
			nickname, phone, email, avatar, time.Now(), user.ID,
		)
		user.Nickname = nickname
		user.Phone = phone
		user.Email = email
		user.Avatar = avatar
		return &user, nil
	}

	// 2. 未绑定：尝试按手机号匹配已有用户
	phone, _ := uasUser["phone"].(string)
	if phone != "" {
		user, err = scanUserRow(db.QueryRow(
			"SELECT id, username, password, nickname, avatar, phone, email, role, created_at, updated_at FROM users WHERE phone = ?",
			phone,
		))
		if err == nil {
			// 绑定 UAS 用户ID
			_, _ = db.Exec("UPDATE users SET uas_user_id=?, updated_at=? WHERE id=?", uasUniqueID, time.Now(), user.ID)
			return &user, nil
		}
	}

	// 3. 全新用户：自动注册
	nickname, _ := uasUser["nickname"].(string)
	realName, _ := uasUser["real_name"].(string)
	email, _ := uasUser["email"].(string)
	avatar, _ := uasUser["avatar"].(string)
	if nickname == "" {
		nickname = realName
	}
	if nickname == "" {
		nickname = "UAS用户"
	}

	// 生成唯一用户名（基于手机号或UAS ID）
	username := "uas_" + uasUserID
	if phone != "" {
		username = "u" + phone
	}
	// 检查用户名冲突，如有冲突则追加随机数
	var existName string
	if db.QueryRow("SELECT username FROM users WHERE username = ?", username).Scan(&existName) == nil {
		username = username + "_" + uasUserID
	}

	// 生成随机密码（UAS登录用户本地不需要密码）
	randomPwd := genID("pwd")
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte(randomPwd), bcrypt.DefaultCost)

	now := time.Now()
	newUser := models.User{
		ID:        genID("u"),
		Username:  username,
		Password:  string(hashedPwd),
		Nickname:  nickname,
		Avatar:    avatar,
		Phone:     phone,
		Email:     email,
		Role:      "student",
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err = db.Exec(
		"INSERT INTO users (id, username, password, nickname, avatar, phone, email, role, uas_user_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		newUser.ID, newUser.Username, newUser.Password, newUser.Nickname, newUser.Avatar,
		newUser.Phone, newUser.Email, newUser.Role, uasUniqueID, newUser.CreatedAt, newUser.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	return &newUser, nil
}

// redirectToFront 带错误信息重定向到前端登录页
func (h *OAuthHandler) redirectToFront(c *gin.Context, path, errMsg string) {
	// 错误情况下直接跳转到登录页，便于显示错误信息
	target := "/pages/login.html"
	sep := "?"
	if strings.Contains(target, "?") {
		sep = "&"
	}
	target += sep + "uas_error=" + url.QueryEscape(errMsg)
	c.Redirect(http.StatusFound, target)
}
