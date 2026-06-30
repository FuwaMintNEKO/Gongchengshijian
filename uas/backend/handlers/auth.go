package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"uas/config"
	"uas/middleware"
	"uas/models"
	"uas/store"
	"uas/utils"

	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler 认证Handler
type AuthHandler struct {
	store *store.Store
	cfg   *config.Config
}

// NewAuthHandler 创建AuthHandler
func NewAuthHandler(s *store.Store, cfg *config.Config) *AuthHandler {
	return &AuthHandler{store: s, cfg: cfg}
}

// captchaStore 验证码存储（使用base64Captcha自带的内存store，过期10分钟）
var captchaStore = base64Captcha.DefaultMemStore

// verifyCaptcha 校验验证码
func verifyCaptcha(uuid, code string) bool {
	if uuid == "" || code == "" {
		return false
	}
	return captchaStore.Verify(uuid, code, true)
}

// Login 统一登录入口
// POST /api/auth/login
// 支持三类用户登录：
//   - admin（默认）：系统管理员，查 sys_user 表，登录后进管理后台
//   - personal：个体用户（自然人），查 u_user 表，用手机号登录
//   - corp：企业用户（法人），查 u_corp_user 表，用username登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Code     string `json:"code"`     // 验证码
		UUID     string `json:"uuid"`     // 验证码ID
		UserType string `json:"userType"` // admin / personal / corp，默认 admin
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	if req.UserType == "" {
		req.UserType = "admin"
	}

	// 校验验证码（图形验证码，防止暴力登录）
	if !verifyCaptcha(req.UUID, req.Code) {
		utils.BadRequest(c, "验证码错误或已过期")
		return
	}

	db := h.store.GetDB()
	if db == nil {
		utils.Error(c, "数据库未连接")
		return
	}

	// 根据用户类型走不同登录流程
	if req.UserType == "personal" || req.UserType == "corp" {
		h.loginUASUser(c, db, req.Username, req.Password, req.UserType)
		return
	}

	// 默认管理员登录
	h.loginSysUser(c, db, req.Username, req.Password)
}

// loginSysUser 管理员登录（查 sys_user 表）
func (h *AuthHandler) loginSysUser(c *gin.Context, db *sql.DB, username, password string) {
	// 查询用户
	var user models.SysUser
	var passwordHash string
	var nickname sql.NullString
	err := db.QueryRow(
		"SELECT id, username, password, nickname, status FROM sys_user WHERE username = ? AND del_flag = 0",
		username,
	).Scan(&user.ID, &user.Username, &passwordHash, &nickname, &user.Status)
	user.Nickname = nickname.String

	if err == sql.ErrNoRows {
		h.recordLoginLog(nil, username, "password", c.ClientIP(), getUserAgent(c), 0, "用户不存在")
		utils.Error(c, "用户名或密码错误")
		return
	}
	if err != nil {
		utils.Error(c, "查询失败: "+err.Error())
		return
	}

	// 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		h.recordLoginLog(&user.ID, username, "password", c.ClientIP(), getUserAgent(c), 0, "密码错误")
		utils.Error(c, "用户名或密码错误")
		return
	}

	// 校验状态
	if user.Status != 1 {
		h.recordLoginLog(&user.ID, username, "password", c.ClientIP(), getUserAgent(c), 0, "账号已禁用")
		utils.Error(c, "账号已禁用，请联系管理员")
		return
	}

	// 查询角色
	roleKey := "common"
	var roleKeys []string
	rows, _ := db.Query(
		"SELECT r.role_key FROM sys_role r INNER JOIN sys_user_role ur ON r.id = ur.role_id WHERE ur.user_id = ? AND r.del_flag = 0",
		user.ID,
	)
	if rows != nil {
		for rows.Next() {
			var k string
			rows.Scan(&k)
			roleKeys = append(roleKeys, k)
		}
		rows.Close()
	}
	for _, k := range roleKeys {
		if k == "admin" {
			roleKey = "admin"
			break
		}
	}

	// 生成token
	token, err := utils.GenerateToken(user.ID, user.Username, user.Nickname, roleKey, h.cfg.JWT.Secret, h.cfg.JWT.ExpireHours)
	if err != nil {
		utils.Error(c, "Token生成失败")
		return
	}

	// 记录登录日志
	h.recordLoginLog(&user.ID, user.Username, "password", c.ClientIP(), getUserAgent(c), 1, "")

	utils.Success(c, gin.H{
		"token":    token,
		"username": user.Username,
		"nickname": user.Nickname,
		"role":     roleKey,
		"userType": "admin",
	})
}

// loginUASUser 个体/企业用户登录（查 u_user / u_corp_user 表）
func (h *AuthHandler) loginUASUser(c *gin.Context, db *sql.DB, username, password, userType string) {
	var (
		userID       int64
		passwordHash string
		status       int
		nickname     string
	)

	if userType == "personal" {
		// 个体用户用手机号登录
		err := db.QueryRow(
			"SELECT id, password, status, COALESCE(nickname, phone) FROM u_user WHERE phone = ? AND del_flag = 0",
			username,
		).Scan(&userID, &passwordHash, &status, &nickname)
		if err == sql.ErrNoRows {
			h.recordLoginLog(nil, username, "password", c.ClientIP(), getUserAgent(c), 0, "用户不存在")
			utils.Error(c, "账号或密码错误")
			return
		}
		if err != nil {
			utils.Error(c, "查询失败: "+err.Error())
			return
		}
	} else {
		// 企业用户用username登录
		var corpName string
		err := db.QueryRow(
			"SELECT id, password, status, COALESCE(corp_name, username) FROM u_corp_user WHERE username = ? AND del_flag = 0",
			username,
		).Scan(&userID, &passwordHash, &status, &corpName)
		if err == sql.ErrNoRows {
			h.recordLoginLog(nil, username, "password", c.ClientIP(), getUserAgent(c), 0, "用户不存在")
			utils.Error(c, "账号或密码错误")
			return
		}
		if err != nil {
			utils.Error(c, "查询失败: "+err.Error())
			return
		}
		nickname = corpName
	}

	// 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		h.recordLoginLog(&userID, username, "password", c.ClientIP(), getUserAgent(c), 0, "密码错误")
		utils.Error(c, "账号或密码错误")
		return
	}

	// 校验状态
	if status != 1 {
		h.recordLoginLog(&userID, username, "password", c.ClientIP(), getUserAgent(c), 0, "账号已禁用")
		utils.Error(c, "账号已禁用，请联系管理员")
		return
	}

	// 生成token，role=uas_user（前端据此限制菜单）
	roleKey := "uas_user"
	if userType == "corp" {
		roleKey = "uas_corp"
	}
	token, err := utils.GenerateToken(userID, username, nickname, roleKey, h.cfg.JWT.Secret, h.cfg.JWT.ExpireHours)
	if err != nil {
		utils.Error(c, "Token生成失败")
		return
	}

	// 记录登录日志
	h.recordLoginLog(&userID, username, "password", c.ClientIP(), getUserAgent(c), 1, "")

	utils.Success(c, gin.H{
		"token":    token,
		"username": username,
		"nickname": nickname,
		"role":     roleKey,
		"userType": userType,
	})
}

// Logout 退出登录
// POST /api/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// 记录退出审计日志
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	h.recordAuditLog(username, "退出", "管理员退出登录", c.ClientIP())
	_ = userID
	utils.Success(c, nil)
}

// GetUserInfo 获取当前登录用户信息
// GET /api/auth/userinfo
func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		utils.Unauthorized(c, "未登录")
		return
	}

	db := h.store.GetDB()
	var user models.SysUser
	var nickname, email, phone, avatar sql.NullString
	err := db.QueryRow(
		"SELECT id, username, nickname, email, phone, sex, avatar, status, create_time FROM sys_user WHERE id = ?",
		userID,
	).Scan(&user.ID, &user.Username, &nickname, &email, &phone, &user.Sex, &avatar, &user.Status, &user.CreateTime)
	user.Nickname = nickname.String
	user.Email = email.String
	user.Phone = phone.String
	user.Avatar = avatar.String

	if err != nil {
		utils.Error(c, "用户不存在")
		return
	}

	// 查询角色
	var roles []string
	rows, _ := db.Query(
		"SELECT r.role_key FROM sys_role r INNER JOIN sys_user_role ur ON r.id = ur.role_id WHERE ur.user_id = ?",
		userID,
	)
	if rows != nil {
		for rows.Next() {
			var k string
			rows.Scan(&k)
			roles = append(roles, k)
		}
		rows.Close()
	}
	if len(roles) == 0 {
		roles = []string{"common"}
	}

	// 查询权限菜单
	var perms []string
	rows, _ = db.Query(
		`SELECT DISTINCT m.perms FROM sys_menu m
		 INNER JOIN sys_role_menu rm ON m.id = rm.menu_id
		 INNER JOIN sys_user_role ur ON rm.role_id = ur.role_id
		 WHERE ur.user_id = ? AND m.perms != ''`,
		userID,
	)
	if rows != nil {
		for rows.Next() {
			var p string
			rows.Scan(&p)
			perms = append(perms, p)
		}
		rows.Close()
	}

	utils.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"nickname": user.Nickname,
		"email":    user.Email,
		"phone":    user.Phone,
		"avatar":   user.Avatar,
		"roles":    roles,
		"perms":    perms,
	})
}

// GetRouters 获取当前用户的路由菜单（用于前端动态路由）
// GET /api/auth/routers
func (h *AuthHandler) GetRouters(c *gin.Context) {
	userID := middleware.GetUserID(c)
	db := h.store.GetDB()

	// 查询当前用户有权限的所有菜单
	rows, err := db.Query(
		`SELECT m.id, m.menu_name, m.parent_id, m.menu_sort, m.path, m.component, m.menu_type, m.visible, m.perms, m.icon
		 FROM sys_menu m
		 INNER JOIN sys_role_menu rm ON m.id = rm.menu_id
		 INNER JOIN sys_user_role ur ON rm.role_id = ur.role_id
		 WHERE ur.user_id = ? AND m.menu_type IN ('M', 'C') AND m.visible = 1
		 GROUP BY m.id
		 ORDER BY m.parent_id, m.menu_sort`,
		userID,
	)
	if err != nil {
		utils.Error(c, "查询菜单失败")
		return
	}
	defer rows.Close()

	var menus []models.SysMenu
	for rows.Next() {
		var m models.SysMenu
		var component, perms, icon sql.NullString
		if err := rows.Scan(&m.ID, &m.MenuName, &m.ParentID, &m.MenuSort, &m.Path, &component, &m.MenuType, &m.Visible, &perms, &icon); err != nil {
			continue
		}
		m.Component = component.String
		m.Perms = perms.String
		m.Icon = icon.String
		menus = append(menus, m)
	}

	// 构建菜单树
	tree := buildMenuTree(menus, 0)
	utils.Success(c, tree)
}

// buildMenuTree 构建菜单树
func buildMenuTree(menus []models.SysMenu, parentID int64) []models.SysMenu {
	var tree []models.SysMenu
	for _, m := range menus {
		if m.ParentID == parentID {
			m.Children = buildMenuTree(menus, m.ID)
			tree = append(tree, m)
		}
	}
	return tree
}

// recordLoginLog 记录登录日志
func (h *AuthHandler) recordLoginLog(userID *int64, username, loginType, ip, userAgent string, result int, reason string) {
	db := h.store.GetDB()
	if db == nil {
		return
	}
	_, _ = db.Exec(
		"INSERT INTO u_login_log (user_id, username, login_type, login_ip, login_result, fail_reason, user_agent, login_time) VALUES (?, ?, ?, ?, ?, ?, ?, NOW())",
		userID, username, loginType, ip, result, reason, userAgent,
	)
}

// getUserAgent 从请求头提取 User-Agent
func getUserAgent(c *gin.Context) string {
	return c.GetHeader("User-Agent")
}

// recordAuditLog 记录审计日志
func (h *AuthHandler) recordAuditLog(operName, operType, operContent, ip string) {
	db := h.store.GetDB()
	if db == nil {
		return
	}
	_, _ = db.Exec(
		"INSERT INTO sys_audit_log (oper_name, oper_type, oper_content, oper_ip, oper_time) VALUES (?, ?, ?, ?, NOW())",
		operName, operType, operContent, ip,
	)
}

// GetCaptcha 生成图形验证码
// GET /api/auth/captcha
// 返回base64编码的PNG验证码图片，前端直接用img src展示
func (h *AuthHandler) GetCaptcha(c *gin.Context) {
	// 使用数字+字母混合验证码，6位字符，更易识别
	driver := base64Captcha.NewDriverString(
		60,  // 高度
		160, // 宽度
		0,   // 噪点数量
		2,   // 弯曲程度
		6,   // 字符数
		"abcdefghijklmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXY3456789", // 字符集（去除易混淆的0/O/1/I/2/Z）
		nil, nil, nil,
	)

	captcha := base64Captcha.NewCaptcha(driver, captchaStore)
	id, b64s, _, err := captcha.Generate()
	if err != nil {
		utils.Error(c, "验证码生成失败: "+err.Error())
		return
	}

	utils.Success(c, gin.H{
		"uuid":   id,
		"img":    b64s, // data:image/png;base64,... 格式，前端直接放到img src即可
		"enable": true,
	})
}

// HealthCheck 健康检查
func (h *AuthHandler) HealthCheck(c *gin.Context) {
	dbOK := h.store.Ping() == nil
	status := "ok"
	if !dbOK {
		status = "degraded"
	}
	c.JSON(http.StatusOK, gin.H{
		"status":   status,
		"database": dbOK,
		"time":     time.Now().Format(time.RFC3339),
	})
}
