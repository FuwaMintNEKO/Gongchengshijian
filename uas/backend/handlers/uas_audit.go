package handlers

import (
	"strconv"

	"uas/store"
	"uas/utils"

	"github.com/gin-gonic/gin"
)

// AuditHandler 审核管理
type AuditHandler struct {
	store *store.Store
}

func NewAuditHandler(s *store.Store) *AuditHandler {
	return &AuditHandler{store: s}
}

// List 审核列表（UNION 自然人+法人，统一分页）
func (h *AuditHandler) List(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	offset := (page - 1) * pageSize
	userType := c.DefaultQuery("userType", "all") // all/personal/corp
	auditStatus := c.Query("auditStatus")

	db := h.store.GetDB()

	type AuditItem struct {
		ID          int64  `json:"id"`
		UserType    string `json:"userType"`
		Username    string `json:"username"`
		RealName    string `json:"realName"`
		Phone       string `json:"phone"`
		AuditStatus int    `json:"auditStatus"`
		AuditRemark string `json:"auditRemark"`
		CreateTime  string `json:"createTime"`
	}

	// 构建 UNION ALL 查询
	// 自然人: audit_status IN (1) = 待审核, audit_status IN (2) = 已通过
	var filters []string
	var args []interface{}

	// userType 过滤
	if userType == "personal" {
		filters = append(filters, " AND source = 'personal'")
	} else if userType == "corp" {
		filters = append(filters, " AND source = 'corp'")
	}

	// auditStatus 过滤: 空=全部待审，0=未提交，1=待审核，2=通过，3=驳回
	if auditStatus != "" {
		filters = append(filters, " AND audit_status = ?")
		args = append(args, auditStatus)
	} else {
		// 默认只显示待审核(audit_status=1)和驳回(audit_status=3)
		filters = append(filters, " AND audit_status IN (1, 3)")
	}

	filterSQL := ""
	for _, f := range filters {
		filterSQL += f
	}

	subQuery := `
		SELECT id, 'personal' AS source, phone AS username, COALESCE(real_name, phone) AS real_name,
		       phone, audit_status, COALESCE(audit_remark, '') AS audit_remark,
		       create_time
		FROM u_user
		WHERE del_flag = 0 AND audit_status > 0` + filterSQL + `
		UNION ALL
		SELECT id, 'corp' AS source, username, COALESCE(corp_name, username) AS real_name,
		       phone, audit_status, COALESCE(audit_remark, '') AS audit_remark,
		       create_time
		FROM u_corp_user
		WHERE del_flag = 0 AND audit_status > 0` + filterSQL

	// 计数
	var total int64
	countSQL := "SELECT COUNT(*) FROM (" + subQuery + ") AS t"
	countArgs := append(append([]interface{}{}, args...), args...)
	db.QueryRow(countSQL, countArgs...).Scan(&total)

	// 分页查询
	querySQL := "SELECT * FROM (" + subQuery + ") AS t ORDER BY id DESC LIMIT ? OFFSET ?"
	queryArgs := append(append(append([]interface{}{}, args...), args...), pageSize, offset)
	rows, err := db.Query(querySQL, queryArgs...)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}
	defer rows.Close()

	var list []AuditItem
	for rows.Next() {
		var u AuditItem
		if err := rows.Scan(&u.ID, &u.UserType, &u.Username, &u.RealName, &u.Phone, &u.AuditStatus, &u.AuditRemark, &u.CreateTime); err != nil {
			continue
		}
		list = append(list, u)
	}
	if list == nil {
		list = []AuditItem{}
	}
	utils.SuccessPage(c, total, list)
}

// AuditUser 审核自然人用户
func (h *AuditHandler) AuditUser(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		AuditStatus int    `json:"auditStatus"` // 2-通过 3-驳回
		AuditRemark string `json:"auditRemark"`
	}
	c.ShouldBindJSON(&req)

	if req.AuditStatus != 2 && req.AuditStatus != 3 {
		utils.BadRequest(c, "审核状态非法")
		return
	}

	db := h.store.GetDB()
	_, err := db.Exec(
		"UPDATE u_user SET audit_status = ?, audit_remark = ?, audit_time = NOW() WHERE id = ?",
		req.AuditStatus, req.AuditRemark, id,
	)
	if err != nil {
		utils.Error(c, "审核失败")
		return
	}
	utils.SuccessMsg(c, "审核成功", nil)
}

// AuditCorp 审核法人用户
func (h *AuditHandler) AuditCorp(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		AuditStatus int    `json:"auditStatus"`
		AuditRemark string `json:"auditRemark"`
	}
	c.ShouldBindJSON(&req)

	if req.AuditStatus != 2 && req.AuditStatus != 3 {
		utils.BadRequest(c, "审核状态非法")
		return
	}

	db := h.store.GetDB()
	_, err := db.Exec(
		"UPDATE u_corp_user SET audit_status = ?, audit_remark = ?, audit_time = NOW() WHERE id = ?",
		req.AuditStatus, req.AuditRemark, id,
	)
	if err != nil {
		utils.Error(c, "审核失败")
		return
	}
	utils.SuccessMsg(c, "审核成功", nil)
}
