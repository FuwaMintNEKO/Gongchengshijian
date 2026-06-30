package handlers

import (
	"database/sql"
	"strings"
	"uas/store"
	"uas/utils"

	"github.com/gin-gonic/gin"
)

// LogHandler 日志管理
type LogHandler struct {
	store *store.Store
}

func NewLogHandler(s *store.Store) *LogHandler {
	return &LogHandler{store: s}
}

// LoginLogList 登录日志列表
func (h *LogHandler) LoginLogList(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	offset := (page - 1) * pageSize
	// 前端传 ipaddr / status，兼容后端参数
	username := c.Query("username")
	loginIP := c.Query("ipaddr") // VUE 前端参数名为 ipaddr
	loginResult := c.Query("status")

	db := h.store.GetDB()
	where := "WHERE 1=1"
	args := []interface{}{}
	if username != "" {
		where += " AND username LIKE ?"
		args = append(args, "%"+username+"%")
	}
	if loginIP != "" {
		where += " AND login_ip LIKE ?"
		args = append(args, "%"+loginIP+"%")
	}
	if loginResult != "" {
		where += " AND login_result = ?"
		args = append(args, loginResult)
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM u_login_log "+where, args...).Scan(&total)

	rows, err := db.Query(
		"SELECT id, user_id, username, login_type, login_ip, login_result, COALESCE(fail_reason,''), COALESCE(user_agent,''), login_time FROM u_login_log "+
			where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}
	defer rows.Close()

	// 字段名与 VUE 前端保持一致：ipaddr / status / loginLocation / browser / os / msg
	type LoginLogItem struct {
		ID            int64  `json:"id"`
		UserID        *int64 `json:"userId"`
		Username      string `json:"username"`
		LoginType     string `json:"loginType"`
		Ipaddr        string `json:"ipaddr"`
		Status        int    `json:"status"`
		Msg           string `json:"msg"`
		UserAgent     string `json:"userAgent"`
		Browser       string `json:"browser"`
		Os            string `json:"os"`
		LoginLocation string `json:"loginLocation"`
		LoginTime     string `json:"loginTime"`
	}

	var list []LoginLogItem
	for rows.Next() {
		var l LoginLogItem
		var userAgent, failReason string
		if err := rows.Scan(&l.ID, &l.UserID, &l.Username, &l.LoginType, &l.Ipaddr, &l.Status, &failReason, &userAgent, &l.LoginTime); err != nil {
			continue
		}
		l.UserAgent = userAgent
		// msg：成功为空，失败显示原因
		if l.Status == 0 && failReason != "" {
			l.Msg = failReason
		}
		// 从 UserAgent 解析浏览器和操作系统
		l.Browser, l.Os = parseUserAgent(userAgent)
		// IP 地点映射（简单版本，完整版可接 GeoIP 库）
		l.LoginLocation = ipToLocation(l.Ipaddr)
		list = append(list, l)
	}
	if list == nil {
		list = []LoginLogItem{}
	}
	utils.SuccessPage(c, total, list)
}

// parseUserAgent 从 UA 字符串解析浏览器和操作系统
func parseUserAgent(ua string) (browser, os string) {
	if ua == "" {
		return "", ""
	}
	uaLower := strings.ToLower(ua)
	// 浏览器
	switch {
	case strings.Contains(uaLower, "edg"):
		browser = "Edge"
	case strings.Contains(uaLower, "chrome"):
		browser = "Chrome"
	case strings.Contains(uaLower, "firefox"):
		browser = "Firefox"
	case strings.Contains(uaLower, "safari") && !strings.Contains(uaLower, "chrome"):
		browser = "Safari"
	default:
		browser = ""
	}
	// 操作系统
	switch {
	case strings.Contains(uaLower, "windows nt 10"):
		os = "Windows 10"
	case strings.Contains(uaLower, "windows nt 6.3"):
		os = "Windows 8.1"
	case strings.Contains(uaLower, "windows nt 6.1"):
		os = "Windows 7"
	case strings.Contains(uaLower, "windows"):
		os = "Windows"
	case strings.Contains(uaLower, "mac os x") || strings.Contains(uaLower, "macintosh"):
		os = "Mac OS"
	case strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "ipad"):
		os = "iOS"
	case strings.Contains(uaLower, "android"):
		os = "Android"
	case strings.Contains(uaLower, "linux"):
		os = "Linux"
	}
	return
}

// ipToLocation IP地址转换为登录地点（简单版，内网/本机返回对应文本）
func ipToLocation(ip string) string {
	if ip == "" {
		return ""
	}
	if ip == "127.0.0.1" || ip == "::1" || strings.HasPrefix(ip, "0:") {
		return "内网IP"
	}
	if strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "172.16.") || strings.HasPrefix(ip, "172.17.") || strings.HasPrefix(ip, "172.18.") || strings.HasPrefix(ip, "172.19.") || strings.HasPrefix(ip, "172.2") || strings.HasPrefix(ip, "172.30.") || strings.HasPrefix(ip, "172.31.") {
		return "内网IP"
	}
	return ip
}

func (h *LogHandler) CleanLoginLog(c *gin.Context) {
	db := h.store.GetDB()
	_, err := db.Exec("TRUNCATE TABLE u_login_log")
	if err != nil {
		utils.Error(c, "清空失败")
		return
	}
	utils.SuccessMsg(c, "清空成功", nil)
}

// AuditLogList 审计日志列表
func (h *LogHandler) AuditLogList(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	offset := (page - 1) * pageSize
	operName := c.Query("operName")
	operType := c.Query("operType")

	db := h.store.GetDB()
	where := "WHERE 1=1"
	args := []interface{}{}
	if operName != "" {
		where += " AND oper_name LIKE ?"
		args = append(args, "%"+operName+"%")
	}
	if operType != "" {
		where += " AND oper_type LIKE ?"
		args = append(args, "%"+operType+"%")
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM sys_audit_log "+where, args...).Scan(&total)

	rows, err := db.Query(
		"SELECT id, oper_name, oper_type, oper_content, oper_ip, oper_time FROM sys_audit_log "+
			where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}
	defer rows.Close()

	type AuditLogItem struct {
		ID          int64  `json:"id"`
		OperName    string `json:"operName"`
		OperType    string `json:"operType"`
		OperContent string `json:"operContent"`
		OperIP      string `json:"operIp"`
		OperTime    string `json:"operTime"`
	}
	var list []AuditLogItem
	for rows.Next() {
		var a AuditLogItem
		var operName, operType, operContent, operIP sql.NullString
		if err := rows.Scan(&a.ID, &operName, &operType, &operContent, &operIP, &a.OperTime); err != nil {
			continue
		}
		a.OperName = operName.String
		a.OperType = operType.String
		a.OperContent = operContent.String
		a.OperIP = operIP.String
		list = append(list, a)
	}
	if list == nil {
		list = []AuditLogItem{}
	}
	utils.SuccessPage(c, total, list)
}

func (h *LogHandler) CleanAuditLog(c *gin.Context) {
	db := h.store.GetDB()
	_, err := db.Exec("TRUNCATE TABLE sys_audit_log")
	if err != nil {
		utils.Error(c, "清空失败")
		return
	}
	utils.SuccessMsg(c, "清空成功", nil)
}

// SmsLogList 短信日志列表
func (h *LogHandler) SmsLogList(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	offset := (page - 1) * pageSize
	phone := c.Query("phone")

	db := h.store.GetDB()
	where := "WHERE 1=1"
	args := []interface{}{}
	if phone != "" {
		where += " AND phone LIKE ?"
		args = append(args, "%"+phone+"%")
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM sys_sms_log "+where, args...).Scan(&total)

	rows, err := db.Query(
		"SELECT id, phone, template, content, send_result, send_time FROM sys_sms_log "+
			where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}
	defer rows.Close()

	type SmsLogItem struct {
		ID         int64  `json:"id"`
		Phone      string `json:"phone"`
		Template   string `json:"template"`
		Content    string `json:"content"`
		SendResult string `json:"sendResult"`
		SendTime   string `json:"sendTime"`
	}
	var list []SmsLogItem
	for rows.Next() {
		var s SmsLogItem
		var template, content, sendResult sql.NullString
		if err := rows.Scan(&s.ID, &s.Phone, &template, &content, &sendResult, &s.SendTime); err != nil {
			continue
		}
		s.Template = template.String
		s.Content = content.String
		s.SendResult = sendResult.String
		list = append(list, s)
	}
	if list == nil {
		list = []SmsLogItem{}
	}
	utils.SuccessPage(c, total, list)
}

func (h *LogHandler) CleanSmsLog(c *gin.Context) {
	db := h.store.GetDB()
	_, err := db.Exec("TRUNCATE TABLE sys_sms_log")
	if err != nil {
		utils.Error(c, "清空失败")
		return
	}
	utils.SuccessMsg(c, "清空成功", nil)
}

// OperLogList 操作日志列表（基于审计日志）
func (h *LogHandler) OperLogList(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	offset := (page - 1) * pageSize
	operName := c.Query("operName")
	module := c.Query("module")
	operType := c.Query("operType")

	db := h.store.GetDB()
	where := "WHERE 1=1"
	args := []interface{}{}
	if operName != "" {
		where += " AND oper_name LIKE ?"
		args = append(args, "%"+operName+"%")
	}
	if module != "" {
		where += " AND oper_type LIKE ?"
		args = append(args, "%"+module+"%")
	}
	if operType != "" {
		where += " AND oper_type LIKE ?"
		args = append(args, "%"+operType+"%")
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM sys_audit_log "+where, args...).Scan(&total)

	rows, err := db.Query(
		"SELECT id, oper_name, oper_type, oper_content, oper_ip, oper_time FROM sys_audit_log "+
			where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}
	defer rows.Close()

	type OperLogItem struct {
		ID            int64  `json:"id"`
		OperName      string `json:"operName"`
		Module        string `json:"module"`
		OperType      string `json:"operType"`
		Description   string `json:"description"`
		RequestMethod string `json:"requestMethod"`
		OperIP        string `json:"operIp"`
		CostTime      int    `json:"costTime"`
		OperTime      string `json:"operTime"`
		OperUrl       string `json:"operUrl"`
		OperParam     string `json:"operParam"`
		JsonResult    string `json:"jsonResult"`
	}
	var list []OperLogItem
	for rows.Next() {
		var l OperLogItem
		var content string
		rows.Scan(&l.ID, &l.OperName, &l.OperType, &content, &l.OperIP, &l.OperTime)
		l.Description = content
		l.Module = l.OperType
		list = append(list, l)
	}
	if list == nil {
		list = []OperLogItem{}
	}
	utils.SuccessPage(c, total, list)
}

func (h *LogHandler) CleanOperLog(c *gin.Context) {
	db := h.store.GetDB()
	_, err := db.Exec("TRUNCATE TABLE sys_audit_log")
	if err != nil {
		utils.Error(c, "清空失败")
		return
	}
	utils.SuccessMsg(c, "清空成功", nil)
}
