package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"uas/config"
	"uas/handlers"
	"uas/middleware"
	"uas/store"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	fmt.Printf("[UAS] 启动中... 端口=%s, DB=%s:%s/%s\n", cfg.ServerPort, cfg.DB.Host, cfg.DB.Port, cfg.DB.DBName)

	dbStore := store.NewStore(&cfg.DB)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 无缓存
	r.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Next()
	})

	authHandler := handlers.NewAuthHandler(dbStore, cfg)
	userHandler := handlers.NewUASUserHandler(dbStore)
	corpHandler := handlers.NewUASCorpHandler(dbStore)
	auditHandler := handlers.NewAuditHandler(dbStore)
	appHandler := handlers.NewAppHandler(dbStore, cfg)
	grantHandler := handlers.NewGrantHandler(dbStore)
	sysUserHandler := handlers.NewSysUserHandler(dbStore)
	roleHandler := handlers.NewRoleHandler(dbStore)
	menuHandler := handlers.NewMenuHandler(dbStore)
	registerHandler := handlers.NewRegisterHandler(dbStore, cfg)
	statHandler := handlers.NewStatHandler(dbStore)
	logHandler := handlers.NewLogHandler(dbStore)
	oauthHandler := handlers.NewOAuthHandler(dbStore, cfg)
	profileHandler := handlers.NewProfileHandler(dbStore, cfg)

	api := r.Group("/api")
	{
		// 健康检查
		api.GET("/health", authHandler.HealthCheck)

		// 登录认证（免鉴权）
		api.GET("/auth/captcha", authHandler.GetCaptcha)
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/logout", authHandler.Logout)

		// 用户注册（免鉴权）- 自然人用户自助注册UAS账号
		api.POST("/auth/register", registerHandler.Register)
		api.GET("/auth/check-phone", registerHandler.CheckPhone)

		// OAuth2 端点（免鉴权，按OAuth2规范校验）
		api.GET("/oauth/authorize", oauthHandler.Authorize)
		api.POST("/oauth/authorize", oauthHandler.AuthorizeConfirm)
		api.POST("/oauth/token", oauthHandler.Token)
		api.GET("/oauth/userinfo", oauthHandler.UserInfo)
		// UAS用户登录（OAuth2 流程中使用）
		api.POST("/uas/login", oauthHandler.UASLogin)
		api.GET("/uas/userinfo", oauthHandler.UASUserInfo)
	}

	// 以下接口需要登录
	auth := api.Group("")
	auth.Use(middleware.JWTAuth(cfg.JWT.Secret))
	{
		// 当前用户
		auth.GET("/auth/userinfo", authHandler.GetUserInfo)
		auth.GET("/auth/routers", authHandler.GetRouters)
		auth.GET("/auth/profile", profileHandler.GetProfile)
		auth.PUT("/auth/profile", profileHandler.UpdateProfile)
		auth.PUT("/auth/password", profileHandler.ChangePassword)
		auth.POST("/auth/avatar", profileHandler.UploadAvatar)

		// 用户管理 - 自然人用户
		auth.GET("/uas/user/list", userHandler.List)
		auth.GET("/uas/user/:id", userHandler.Get)
		auth.POST("/uas/user", userHandler.Create)
		auth.PUT("/uas/user", userHandler.Update)
		auth.DELETE("/uas/user/:id", userHandler.Delete)
		auth.PUT("/uas/user/:id/status", userHandler.ChangeStatus)

		// 用户管理 - 法人用户
		auth.GET("/uas/corp/list", corpHandler.List)
		auth.GET("/uas/corp/:id", corpHandler.Get)
		auth.POST("/uas/corp", corpHandler.Create)
		auth.PUT("/uas/corp", corpHandler.Update)
		auth.DELETE("/uas/corp/:id", corpHandler.Delete)
		auth.PUT("/uas/corp/:id/status", corpHandler.ChangeStatus)

		// 用户管理 - 审核管理
		auth.GET("/uas/audit/list", auditHandler.List)
		auth.PUT("/uas/audit/user/:id", auditHandler.AuditUser)
		auth.PUT("/uas/audit/corp/:id", auditHandler.AuditCorp)

		// 应用接入 - 应用管理
		auth.GET("/uas/app/list", appHandler.List)
		auth.GET("/uas/app/:id", appHandler.Get)
		auth.POST("/uas/app", appHandler.Create)
		auth.PUT("/uas/app", appHandler.Update)
		auth.DELETE("/uas/app/:id", appHandler.Delete)
		auth.PUT("/uas/app/:id/resetSecret", appHandler.ResetSecret)

		// 应用接入 - 授权管理
		auth.GET("/uas/grant/list", grantHandler.List)
		auth.DELETE("/uas/grant/:id", grantHandler.Delete)

		// 系统管理 - 管理员
		auth.GET("/system/user/list", sysUserHandler.List)
		auth.GET("/system/user/:id", sysUserHandler.Get)
		auth.POST("/system/user", sysUserHandler.Create)
		auth.PUT("/system/user", sysUserHandler.Update)
		auth.DELETE("/system/user/:id", sysUserHandler.Delete)
		auth.PUT("/system/user/:id/resetPwd", sysUserHandler.ResetPwd)
		auth.PUT("/system/user/:id/status", sysUserHandler.ChangeStatus)

		// 系统管理 - 角色
		auth.GET("/system/role/list", roleHandler.List)
		auth.GET("/system/role/:id", roleHandler.Get)
		auth.POST("/system/role", roleHandler.Create)
		auth.PUT("/system/role", roleHandler.Update)
		auth.DELETE("/system/role/:id", roleHandler.Delete)

		// 系统管理 - 菜单
		auth.GET("/system/menu/list", menuHandler.List)
		auth.GET("/system/menu/:id", menuHandler.Get)
		auth.POST("/system/menu", menuHandler.Create)
		auth.PUT("/system/menu", menuHandler.Update)
		auth.DELETE("/system/menu/:id", menuHandler.Delete)
		auth.GET("/system/menu/treeselect", menuHandler.TreeSelect)
		auth.GET("/system/menu/roleMenuTreeselect/:roleId", menuHandler.RoleMenuTreeSelect)

		// 统计分析
		auth.GET("/stat/account", statHandler.Account)
		auth.GET("/stat/login", statHandler.Login)
		auth.GET("/stat/api", statHandler.API)
		auth.GET("/stat/sms", statHandler.SMS)
		auth.GET("/stat/overview", statHandler.Overview)
		auth.GET("/stat/trend", statHandler.Trend)
		auth.GET("/stat/appType", statHandler.AppType)
		auth.GET("/stat/topApps", statHandler.TopApps)
		auth.GET("/stat/activeUsers", statHandler.ActiveUsers)

		// 日志管理
		auth.GET("/log/loginLog/list", logHandler.LoginLogList)
		auth.DELETE("/log/loginLog/clean", logHandler.CleanLoginLog)
		auth.GET("/log/auditLog/list", logHandler.AuditLogList)
		auth.DELETE("/log/auditLog/clean", logHandler.CleanAuditLog)
		auth.GET("/log/smsLog/list", logHandler.SmsLogList)
		auth.DELETE("/log/smsLog/clean", logHandler.CleanSmsLog)
		auth.GET("/log/operlog/list", logHandler.OperLogList)
		auth.DELETE("/log/operlog/clean", logHandler.CleanOperLog)
		auth.GET("/log/loginlog/list", logHandler.LoginLogList)
	}

	// 静态资源（头像上传目录）
	os.MkdirAll("uploads/avatar", 0755)
	r.Static("/uploads", "./uploads")

	// OAuth 授权页（原生 HTML，无需前端构建）
	// 访问 /oauth/authorize 返回授权页 HTML，页面内 JS 调用 /api/oauth/* 接口
	r.StaticFile("/oauth/authorize", "./public/oauth/authorize.html")

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[UAS] 启动失败: %v", err)
		}
	}()

	fmt.Printf("[UAS] 服务已启动: http://localhost:%s\n", cfg.ServerPort)
	fmt.Printf("[UAS] 默认账号: admin / admin123\n")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("[UAS] 正在关闭...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("[UAS] 关闭失败:", err)
	}
	fmt.Println("[UAS] 已退出")
}
