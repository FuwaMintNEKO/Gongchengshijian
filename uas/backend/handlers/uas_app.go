package handlers

import (
	"database/sql"
	"net/url"
	"strconv"

	"uas/config"
	"uas/store"
	"uas/utils"

	"github.com/gin-gonic/gin"
)

// AppHandler 第三方应用管理
type AppHandler struct {
	store *store.Store
	cfg   *config.Config
}

func NewAppHandler(s *store.Store, cfg *config.Config) *AppHandler {
	return &AppHandler{store: s, cfg: cfg}
}

// List 应用列表
func (h *AppHandler) List(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	offset := (page - 1) * pageSize

	appName := c.Query("appName")
	appType := c.Query("appType")
	status := c.Query("status")

	db := h.store.GetDB()
	where := "WHERE del_flag = 0"
	args := []interface{}{}
	if appName != "" {
		where += " AND app_name LIKE ?"
		args = append(args, "%"+appName+"%")
	}
	if appType != "" {
		where += " AND app_type = ?"
		args = append(args, appType)
	}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM u_app "+where, args...).Scan(&total)

	rows, err := db.Query(
		"SELECT id, app_id, app_name, app_type, sm4_secret, app_secret, redirect_uri, status, description, create_time, update_time FROM u_app "+
			where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}
	defer rows.Close()

	type AppItem struct {
		ID          int64  `json:"id"`
		AppID       string `json:"appId"`
		AppName     string `json:"appName"`
		AppType     string `json:"appType"`
		SM4Secret   string `json:"sm4Secret"`
		AppSecret   string `json:"appSecret"`
		RedirectURI string `json:"redirectUri"`
		Status      int    `json:"status"`
		Description string `json:"description"`
		CreateTime  string `json:"createTime"`
		UpdateTime  string `json:"updateTime"`
	}

	var list []AppItem
	for rows.Next() {
		var a AppItem
		var appType, sm4Secret, redirectURI, description sql.NullString
		if err := rows.Scan(&a.ID, &a.AppID, &a.AppName, &appType, &sm4Secret, &a.AppSecret, &redirectURI, &a.Status, &description, &a.CreateTime, &a.UpdateTime); err != nil {
			continue
		}
		a.AppType = appType.String
		a.SM4Secret = utils.MaskString(sm4Secret.String, 4, 4)
		a.AppSecret = utils.MaskString(a.AppSecret, 4, 4)
		a.RedirectURI = redirectURI.String
		a.Description = description.String
		list = append(list, a)
	}
	if list == nil {
		list = []AppItem{}
	}

	utils.SuccessPage(c, total, list)
}

// Get 应用详情
func (h *AppHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	db := h.store.GetDB()

	var appID, appName, appSecret string
	var appType, sm4Secret, redirectURI, description sql.NullString
	var status int
	var createTime, updateTime string

	err := db.QueryRow(
		"SELECT app_id, app_name, app_type, sm4_secret, app_secret, redirect_uri, status, description, create_time, update_time FROM u_app WHERE id = ? AND del_flag = 0",
		id,
	).Scan(&appID, &appName, &appType, &sm4Secret, &appSecret, &redirectURI, &status, &description, &createTime, &updateTime)

	if err != nil {
		utils.Error(c, "应用不存在")
		return
	}

	utils.Success(c, gin.H{
		"id":          id,
		"appId":       appID,
		"appName":     appName,
		"appType":     appType.String,
		"sm4Secret":   sm4Secret.String,
		"appSecret":   appSecret,
		"redirectUri": redirectURI.String,
		"status":      status,
		"description": description.String,
		"createTime":  createTime,
		"updateTime":  updateTime,
	})
}

// Create 新增应用
func (h *AppHandler) Create(c *gin.Context) {
	var req struct {
		AppName     string `json:"appName" binding:"required"`
		AppType     string `json:"appType"`
		RedirectURI string `json:"redirectUri" binding:"required"`
		Status      int    `json:"status"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "应用名称、回调地址必填")
		return
	}

	if req.AppType == "" {
		req.AppType = "web"
	}
	if req.Status == 0 {
		req.Status = 1
	}

	appID := utils.GenerateAppID()
	appSecret := utils.GenerateAppSecret()
	sm4Secret := utils.GenerateSM4Secret()

	db := h.store.GetDB()
	_, err := db.Exec(
		"INSERT INTO u_app (app_id, app_name, app_type, sm4_secret, app_secret, redirect_uri, status, description) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		appID, req.AppName, req.AppType, sm4Secret, appSecret, req.RedirectURI, req.Status, req.Description,
	)
	if err != nil {
		utils.Error(c, "新增失败: "+err.Error())
		return
	}
	utils.SuccessMsg(c, "新增成功", gin.H{
		"appId":     appID,
		"appSecret": appSecret,
		"sm4Secret": sm4Secret,
	})
}

// Update 修改应用
func (h *AppHandler) Update(c *gin.Context) {
	var req struct {
		ID          int64  `json:"id" binding:"required"`
		AppName     string `json:"appName"`
		AppType     string `json:"appType"`
		RedirectURI string `json:"redirectUri"`
		Status      int    `json:"status"`
		Description string `json:"description"`
	}
	c.ShouldBindJSON(&req)

	db := h.store.GetDB()
	_, err := db.Exec(
		"UPDATE u_app SET app_name=?, app_type=?, redirect_uri=?, status=?, description=? WHERE id=? AND del_flag=0",
		req.AppName, req.AppType, req.RedirectURI, req.Status, req.Description, req.ID,
	)
	if err != nil {
		utils.Error(c, "修改失败")
		return
	}
	utils.SuccessMsg(c, "修改成功", nil)
}

// Delete 删除应用
func (h *AppHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	db := h.store.GetDB()
	_, err := db.Exec("UPDATE u_app SET del_flag = 1 WHERE id = ?", id)
	if err != nil {
		utils.Error(c, "删除失败")
		return
	}
	utils.SuccessMsg(c, "删除成功", nil)
}

// ResetSecret 重置应用密钥
func (h *AppHandler) ResetSecret(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	appSecret := utils.GenerateAppSecret()
	sm4Secret := utils.GenerateSM4Secret()

	db := h.store.GetDB()
	_, err := db.Exec(
		"UPDATE u_app SET app_secret = ?, sm4_secret = ? WHERE id = ? AND del_flag = 0",
		appSecret, sm4Secret, id,
	)
	if err != nil {
		utils.Error(c, "重置失败")
		return
	}
	utils.SuccessMsg(c, "重置成功", gin.H{
		"appSecret": appSecret,
		"sm4Secret": sm4Secret,
	})
}

// VerifyApp 校验应用（内部使用）
func (h *AppHandler) VerifyApp(appID, redirectURI string) (*struct {
	ID          int64
	AppID       string
	AppSecret   string
	SM4Secret   string
	RedirectURI string
	Status      int
}, error) {
	db := h.store.GetDB()
	var id int64
	var appSecret string
	var sm4Secret, dbRedirect sql.NullString
	var status int
	err := db.QueryRow(
		"SELECT id, app_secret, sm4_secret, redirect_uri, status FROM u_app WHERE app_id = ? AND del_flag = 0",
		appID,
	).Scan(&id, &appSecret, &sm4Secret, &dbRedirect, &status)

	if err != nil {
		return nil, err
	}

	// 校验回调地址
	// 策略：只校验 path 部分，允许 host:port 变化（适配同 IP 不同端口部署）
	// 数据库中注册如 http://localhost:8080/oauth/callback，实际请求如 http://1.2.3.4:28080/oauth/callback
	// 只要 path 都是 /oauth/callback 即通过
	dbRedirectStr := dbRedirect.String
	if dbRedirectStr != "" && dbRedirectStr != "*" {
		dbPath := redirectPath(dbRedirectStr)
		reqPath := redirectPath(redirectURI)
		if dbPath == "" || reqPath == "" || dbPath != reqPath {
			return nil, sql.ErrNoRows
		}
	}

	if status != 1 {
		return nil, sql.ErrNoRows
	}

	return &struct {
		ID          int64
		AppID       string
		AppSecret   string
		SM4Secret   string
		RedirectURI string
		Status      int
	}{
		ID:          id,
		AppID:       appID,
		AppSecret:   appSecret,
		SM4Secret:   sm4Secret.String,
		RedirectURI: dbRedirectStr,
		Status:      status,
	}, nil
}

// redirectPath 提取 redirect_uri 的 path 部分（小写 scheme+host 后的路径）
// 例如 http://localhost:8080/oauth/callback -> /oauth/callback
// 解析失败返回空字符串
func redirectPath(rawURI string) string {
	u, err := url.Parse(rawURI)
	if err != nil {
		return ""
	}
	return u.EscapedPath()
}
