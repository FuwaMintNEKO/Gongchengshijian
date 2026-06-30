package handlers

import (
	"database/sql"
	"regexp"
	"strings"
	"uas/config"
	"uas/store"
	"uas/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// RegisterHandler 用户注册Handler
// 提供个体用户自助注册功能，注册后可直接通过OAuth2登录接入的第三方应用
type RegisterHandler struct {
	store *store.Store
	cfg   *config.Config
}

// NewRegisterHandler 创建RegisterHandler
func NewRegisterHandler(s *store.Store, cfg *config.Config) *RegisterHandler {
	return &RegisterHandler{store: s, cfg: cfg}
}

// Register 个体用户注册
// POST /api/auth/register
// 用户通过手机号自助注册UAS账号，注册成功后可直接用于OAuth2登录第三方应用
func (h *RegisterHandler) Register(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone" binding:"required"`
		Password string `json:"password" binding:"required"`
		Nickname string `json:"nickname"`
		RealName string `json:"realName"`
		Email    string `json:"email"`
		Code     string `json:"code"` // 图形验证码
		UUID     string `json:"uuid"` // 图形验证码ID
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "手机号和密码为必填项")
		return
	}

	// 校验验证码（图形验证码，防止批量注册）
	if !verifyCaptcha(req.UUID, req.Code) {
		utils.BadRequest(c, "验证码错误或已过期")
		return
	}

	// 校验手机号格式（11位数字，1开头）
	phoneReg := regexp.MustCompile(`^1[3-9]\d{9}$`)
	if !phoneReg.MatchString(req.Phone) {
		utils.BadRequest(c, "手机号格式不正确")
		return
	}

	// 校验密码强度（至少6位）
	if len(req.Password) < 6 {
		utils.BadRequest(c, "密码至少6位")
		return
	}

	// 校验邮箱格式（如果填了）
	if req.Email != "" {
		emailReg := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailReg.MatchString(req.Email) {
			utils.BadRequest(c, "邮箱格式不正确")
			return
		}
	}

	// 昵称默认用手机号后4位
	if strings.TrimSpace(req.Nickname) == "" {
		req.Nickname = "用户" + req.Phone[len(req.Phone)-4:]
	}

	db := h.store.GetDB()
	if db == nil {
		utils.Error(c, "数据库未连接")
		return
	}

	// 检查手机号是否已注册
	var existID int64
	err := db.QueryRow("SELECT id FROM u_user WHERE phone = ? AND del_flag = 0", req.Phone).Scan(&existID)
	if err == nil {
		utils.Error(c, "该手机号已注册，请直接登录")
		return
	}
	if err != sql.ErrNoRows {
		utils.Error(c, "查询失败: "+err.Error())
		return
	}

	// 加密密码
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.Error(c, "密码加密失败")
		return
	}

	// 插入用户记录
	// - auth_level=L1（基础认证等级，仅手机号验证）
	// - audit_status=2（审核通过：注册即视为L1实名未认证但可用，更高等级需后续提交资料审核）
	// - status=1（启用）
	realName := strings.TrimSpace(req.RealName)
	email := strings.TrimSpace(req.Email)

	result, err := db.Exec(
		"INSERT INTO u_user (phone, password, real_name, id_card_type, id_card_no, auth_level, nickname, email, status, audit_status) VALUES (?, ?, ?, 1, NULL, 'L1', ?, ?, 1, 2)",
		req.Phone, string(hash), realName, req.Nickname, email,
	)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			utils.Error(c, "该手机号已注册，请直接登录")
			return
		}
		utils.Error(c, "注册失败: "+err.Error())
		return
	}

	userID, _ := result.LastInsertId()

	// 自动登录：生成token，省去用户注册后再去登录的步骤
	token, err := utils.GenerateToken(userID, req.Phone, req.Nickname, "uas_user", h.cfg.JWT.Secret, h.cfg.JWT.ExpireHours)
	if err != nil {
		// 注册成功但token生成失败，仍返回成功
		utils.SuccessMsg(c, "注册成功，请登录", gin.H{
			"userId":   userID,
			"phone":    req.Phone,
			"nickname": req.Nickname,
		})
		return
	}

	// 记录注册日志到登录日志表（type=register）
	h.recordRegisterLog(userID, req.Phone, c.ClientIP(), c.GetHeader("User-Agent"))

	utils.SuccessMsg(c, "注册成功", gin.H{
		"token":     token,
		"userId":    userID,
		"phone":     req.Phone,
		"nickname":  req.Nickname,
		"authLevel": "L1",
	})
}

// CheckPhone 检查手机号是否已注册
// GET /api/auth/check-phone?phone=xxx
func (h *RegisterHandler) CheckPhone(c *gin.Context) {
	phone := c.Query("phone")
	if phone == "" {
		utils.BadRequest(c, "手机号不能为空")
		return
	}

	phoneReg := regexp.MustCompile(`^1[3-9]\d{9}$`)
	if !phoneReg.MatchString(phone) {
		utils.BadRequest(c, "手机号格式不正确")
		return
	}

	db := h.store.GetDB()
	var existID int64
	err := db.QueryRow("SELECT id FROM u_user WHERE phone = ? AND del_flag = 0", phone).Scan(&existID)
	if err == sql.ErrNoRows {
		utils.SuccessMsg(c, "手机号可注册", gin.H{"available": true})
		return
	}
	if err != nil {
		utils.Error(c, "查询失败: "+err.Error())
		return
	}

	utils.SuccessMsg(c, "手机号已注册", gin.H{"available": false})
}

// recordRegisterLog 记录注册日志到登录日志表（沿用现有 u_login_log 表，login_type=register）
func (h *RegisterHandler) recordRegisterLog(userID int64, phone, ip, userAgent string) {
	db := h.store.GetDB()
	if db == nil {
		return
	}
	_, _ = db.Exec(
		"INSERT INTO u_login_log (user_id, username, login_type, login_ip, login_result, fail_reason, user_agent, login_time) VALUES (?, ?, 'register', ?, 1, '注册成功', ?, NOW())",
		userID, phone, ip, userAgent,
	)
}
