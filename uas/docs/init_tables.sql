-- ============================================================
-- UAS 统一身份认证系统 - 数据库表结构
-- 数据库: uas_db
-- 字符集: utf8mb4
-- ============================================================

CREATE DATABASE IF NOT EXISTS uas_db DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE uas_db;

-- ============================================================
-- 一、统一认证管理（u_ 前缀，4 张表）
-- ============================================================

-- 1. 自然人用户
DROP TABLE IF EXISTS u_user;
CREATE TABLE u_user (
  id              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  phone           VARCHAR(20)  NOT NULL COMMENT '手机号',
  password        VARCHAR(255) NOT NULL COMMENT '密码（BCrypt）',
  real_name       VARCHAR(50)  DEFAULT NULL COMMENT '姓名',
  id_card_type    TINYINT      DEFAULT 1 COMMENT '证件类型: 1-身份证 2-护照 3-军官证',
  id_card_no      VARCHAR(50)  DEFAULT NULL COMMENT '证件号码',
  auth_level      VARCHAR(10)  DEFAULT 'L1' COMMENT '认证等级: L1/L2/L3',
  avatar          VARCHAR(500) DEFAULT NULL COMMENT '头像URL',
  nickname        VARCHAR(50)  DEFAULT NULL COMMENT '昵称',
  email           VARCHAR(100) DEFAULT NULL COMMENT '邮箱',
  status          TINYINT      DEFAULT 1 COMMENT '状态: 0-禁用 1-启用',
  audit_status    TINYINT      DEFAULT 0 COMMENT '审核状态: 0-未提交 1-待审核 2-通过 3-驳回',
  audit_remark    VARCHAR(500) DEFAULT NULL COMMENT '审核备注',
  audit_time      DATETIME     DEFAULT NULL COMMENT '审核时间',
  create_time     DATETIME     DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  update_time     DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后修改时间',
  del_flag        TINYINT      DEFAULT 0 COMMENT '逻辑删除: 0-未删 1-已删',
  PRIMARY KEY (id),
  UNIQUE KEY uk_phone (phone)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='自然人用户表';

-- 2. 法人用户（企业用户）
DROP TABLE IF EXISTS u_corp_user;
CREATE TABLE u_corp_user (
  id              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  username        VARCHAR(50)  NOT NULL COMMENT '登录用户名',
  password        VARCHAR(255) NOT NULL COMMENT '密码（BCrypt）',
  corp_type       VARCHAR(20)  DEFAULT 'enterprise' COMMENT '法人类型: enterprise-企业 institution-事业单位',
  corp_name       VARCHAR(100) NOT NULL COMMENT '法人名称',
  credit_code     VARCHAR(50)  NOT NULL COMMENT '统一社会信用代码',
  legal_person    VARCHAR(50)  DEFAULT NULL COMMENT '法定代表人',
  legal_id_card   VARCHAR(50)  DEFAULT NULL COMMENT '法定代表人证件号',
  agent_name      VARCHAR(50)  DEFAULT NULL COMMENT '经办人姓名',
  agent_id_card   VARCHAR(50)  DEFAULT NULL COMMENT '经办人证件号',
  phone           VARCHAR(20)  DEFAULT NULL COMMENT '联系电话',
  status          TINYINT      DEFAULT 1 COMMENT '账号状态: 0-禁用 1-启用',
  audit_status    TINYINT      DEFAULT 0 COMMENT '审核状态: 0-未提交 1-待审核 2-通过 3-驳回',
  audit_remark    VARCHAR(500) DEFAULT NULL COMMENT '审核备注',
  audit_time      DATETIME     DEFAULT NULL COMMENT '审核时间',
  create_time     DATETIME     DEFAULT CURRENT_TIMESTAMP COMMENT '注册时间',
  update_time     DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后修改时间',
  del_flag        TINYINT      DEFAULT 0 COMMENT '逻辑删除: 0-未删 1-已删',
  PRIMARY KEY (id),
  UNIQUE KEY uk_username (username),
  UNIQUE KEY uk_credit_code (credit_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='法人用户表';

-- 3. 第三方应用
DROP TABLE IF EXISTS u_app;
CREATE TABLE u_app (
  id              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  app_id          VARCHAR(32)  NOT NULL COMMENT 'AppId（应用唯一标识）',
  app_name        VARCHAR(100) NOT NULL COMMENT '应用名称',
  app_type        VARCHAR(30)  DEFAULT 'web' COMMENT '应用类型: web-Web应用程序 mobile-移动应用',
  sm4_secret      VARCHAR(255) DEFAULT NULL COMMENT 'SM4加密密钥',
  app_secret      VARCHAR(255) NOT NULL COMMENT '应用Secret',
  redirect_uri    VARCHAR(500) DEFAULT NULL COMMENT '回调地址',
  status          TINYINT      DEFAULT 1 COMMENT '状态: 0-禁用 1-启用',
  description     VARCHAR(500) DEFAULT NULL COMMENT '应用描述',
  create_time     DATETIME     DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  update_time     DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改日期',
  del_flag        TINYINT      DEFAULT 0 COMMENT '逻辑删除: 0-未删 1-已删',
  PRIMARY KEY (id),
  UNIQUE KEY uk_app_id (app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='第三方应用表';

-- 4. 登录日志
DROP TABLE IF EXISTS u_login_log;
CREATE TABLE u_login_log (
  id              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  user_id         BIGINT       DEFAULT NULL COMMENT '用户ID',
  username        VARCHAR(50)  DEFAULT NULL COMMENT '用户名',
  login_type      VARCHAR(20)  DEFAULT NULL COMMENT '登录方式: password-密码 sms-短信 corp-企业',
  login_ip        VARCHAR(50)  DEFAULT NULL COMMENT '登录IP',
  login_result    TINYINT      DEFAULT 1 COMMENT '登录结果: 0-失败 1-成功',
  fail_reason     VARCHAR(255) DEFAULT NULL COMMENT '失败原因',
  user_agent      VARCHAR(500) DEFAULT NULL COMMENT '浏览器UA',
  login_time      DATETIME     DEFAULT CURRENT_TIMESTAMP COMMENT '登录时间',
  PRIMARY KEY (id),
  KEY idx_user_id (user_id),
  KEY idx_login_time (login_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='登录日志表';

-- ============================================================
-- 二、系统管理（sys_ 前缀，5 张表）
-- ============================================================

-- 5. 系统管理员
DROP TABLE IF EXISTS sys_user;
CREATE TABLE sys_user (
  id              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  username        VARCHAR(50)  NOT NULL COMMENT '用户名',
  password        VARCHAR(255) NOT NULL COMMENT '密码（BCrypt）',
  nickname        VARCHAR(50)  DEFAULT NULL COMMENT '昵称',
  email           VARCHAR(100) DEFAULT NULL COMMENT '邮箱',
  phone           VARCHAR(20)  DEFAULT NULL COMMENT '手机号',
  sex             TINYINT      DEFAULT 0 COMMENT '性别: 0-未知 1-男 2-女',
  avatar          VARCHAR(500) DEFAULT NULL COMMENT '头像',
  status          TINYINT      DEFAULT 1 COMMENT '状态: 0-禁用 1-启用',
  dept_id         BIGINT       DEFAULT NULL COMMENT '部门ID',
  remark          VARCHAR(500) DEFAULT NULL COMMENT '备注',
  create_time     DATETIME     DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  update_time     DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  del_flag        TINYINT      DEFAULT 0 COMMENT '逻辑删除: 0-未删 1-已删',
  PRIMARY KEY (id),
  UNIQUE KEY uk_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统管理员表';

-- 6. 角色
DROP TABLE IF EXISTS sys_role;
CREATE TABLE sys_role (
  id              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  role_name       VARCHAR(50)  NOT NULL COMMENT '角色名称',
  role_key        VARCHAR(50)  NOT NULL COMMENT '角色标识',
  role_sort       INT          DEFAULT 0 COMMENT '显示顺序',
  status          TINYINT      DEFAULT 1 COMMENT '状态: 0-禁用 1-启用',
  remark          VARCHAR(500) DEFAULT NULL COMMENT '备注',
  create_time     DATETIME     DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  update_time     DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  del_flag        TINYINT      DEFAULT 0 COMMENT '逻辑删除: 0-未删 1-已删',
  PRIMARY KEY (id),
  UNIQUE KEY uk_role_key (role_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色表';

-- 7. 菜单
DROP TABLE IF EXISTS sys_menu;
CREATE TABLE sys_menu (
  id              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  menu_name       VARCHAR(50)  NOT NULL COMMENT '菜单名称',
  parent_id       BIGINT       DEFAULT 0 COMMENT '父菜单ID',
  menu_sort       INT          DEFAULT 0 COMMENT '显示顺序',
  path            VARCHAR(200) DEFAULT NULL COMMENT '路由地址',
  component       VARCHAR(255) DEFAULT NULL COMMENT '组件路径',
  menu_type       VARCHAR(10)  DEFAULT 'C' COMMENT '菜单类型: M-目录 C-菜单 F-按钮',
  visible         TINYINT      DEFAULT 1 COMMENT '是否可见: 0-隐藏 1-显示',
  perms           VARCHAR(100) DEFAULT NULL COMMENT '权限标识',
  icon            VARCHAR(100) DEFAULT NULL COMMENT '菜单图标',
  create_time     DATETIME     DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  update_time     DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='菜单表';

-- 8. 用户-角色关联
DROP TABLE IF EXISTS sys_user_role;
CREATE TABLE sys_user_role (
  user_id         BIGINT       NOT NULL COMMENT '用户ID',
  role_id         BIGINT       NOT NULL COMMENT '角色ID',
  PRIMARY KEY (user_id, role_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户角色关联表';

-- 9. 角色-菜单关联
DROP TABLE IF EXISTS sys_role_menu;
CREATE TABLE sys_role_menu (
  role_id         BIGINT       NOT NULL COMMENT '角色ID',
  menu_id         BIGINT       NOT NULL COMMENT '菜单ID',
  PRIMARY KEY (role_id, menu_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色菜单关联表';

-- ============================================================
-- 三、扩展表（部门、岗位、字典、参数、公告、审计、短信日志、授权）
-- ============================================================

-- 10. 部门
DROP TABLE IF EXISTS sys_dept;
CREATE TABLE sys_dept (
  id              BIGINT       NOT NULL AUTO_INCREMENT,
  parent_id       BIGINT       DEFAULT 0,
  dept_name       VARCHAR(50)  NOT NULL,
  dept_sort       INT          DEFAULT 0,
  leader          VARCHAR(50)  DEFAULT NULL,
  phone           VARCHAR(20)  DEFAULT NULL,
  status          TINYINT      DEFAULT 1,
  create_time     DATETIME     DEFAULT CURRENT_TIMESTAMP,
  update_time     DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  del_flag        TINYINT      DEFAULT 0,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='部门表';

-- 11. 岗位
DROP TABLE IF EXISTS sys_post;
CREATE TABLE sys_post (
  id              BIGINT       NOT NULL AUTO_INCREMENT,
  post_code       VARCHAR(50)  NOT NULL,
  post_name       VARCHAR(50)  NOT NULL,
  post_sort       INT          DEFAULT 0,
  status          TINYINT      DEFAULT 1,
  remark          VARCHAR(500) DEFAULT NULL,
  create_time     DATETIME     DEFAULT CURRENT_TIMESTAMP,
  update_time     DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  del_flag        TINYINT      DEFAULT 0,
  PRIMARY KEY (id),
  UNIQUE KEY uk_post_code (post_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='岗位表';

-- 12. 字典类型
DROP TABLE IF EXISTS sys_dict_type;
CREATE TABLE sys_dict_type (
  id              BIGINT       NOT NULL AUTO_INCREMENT,
  dict_name       VARCHAR(100) NOT NULL,
  dict_type       VARCHAR(100) NOT NULL,
  status          TINYINT      DEFAULT 1,
  remark          VARCHAR(500) DEFAULT NULL,
  create_time     DATETIME     DEFAULT CURRENT_TIMESTAMP,
  update_time     DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_dict_type (dict_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='字典类型表';

-- 13. 字典数据
DROP TABLE IF EXISTS sys_dict_data;
CREATE TABLE sys_dict_data (
  id              BIGINT       NOT NULL AUTO_INCREMENT,
  dict_type       VARCHAR(100) NOT NULL,
  dict_label      VARCHAR(100) NOT NULL,
  dict_value      VARCHAR(100) NOT NULL,
  dict_sort       INT          DEFAULT 0,
  css_class       VARCHAR(100) DEFAULT NULL,
  list_class      VARCHAR(100) DEFAULT NULL,
  is_default      TINYINT      DEFAULT 0,
  status          TINYINT      DEFAULT 1,
  remark          VARCHAR(500) DEFAULT NULL,
  create_time     DATETIME     DEFAULT CURRENT_TIMESTAMP,
  update_time     DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_dict_type (dict_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='字典数据表';

-- 14. 参数配置
DROP TABLE IF EXISTS sys_config;
CREATE TABLE sys_config (
  id              BIGINT       NOT NULL AUTO_INCREMENT,
  config_name     VARCHAR(100) NOT NULL,
  config_key      VARCHAR(100) NOT NULL,
  config_value    VARCHAR(500) DEFAULT NULL,
  config_type     VARCHAR(10)  DEFAULT 'Y',
  remark          VARCHAR(500) DEFAULT NULL,
  create_time     DATETIME     DEFAULT CURRENT_TIMESTAMP,
  update_time     DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_config_key (config_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='参数配置表';

-- 15. 通知公告
DROP TABLE IF EXISTS sys_notice;
CREATE TABLE sys_notice (
  id              BIGINT       NOT NULL AUTO_INCREMENT,
  notice_title    VARCHAR(100) NOT NULL,
  notice_type     TINYINT      NOT NULL COMMENT '1-通知 2-公告',
  notice_content  TEXT,
  status          TINYINT      DEFAULT 1,
  create_by       VARCHAR(50)  DEFAULT NULL,
  create_time     DATETIME     DEFAULT CURRENT_TIMESTAMP,
  update_time     DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通知公告表';

-- 16. 审计日志
DROP TABLE IF EXISTS sys_audit_log;
CREATE TABLE sys_audit_log (
  id              BIGINT       NOT NULL AUTO_INCREMENT,
  oper_name       VARCHAR(50)  DEFAULT NULL COMMENT '操作人',
  oper_type       VARCHAR(50)  DEFAULT NULL COMMENT '操作类型',
  oper_content    VARCHAR(500) DEFAULT NULL COMMENT '操作内容',
  oper_ip         VARCHAR(50)  DEFAULT NULL COMMENT 'IP地址',
  oper_time       DATETIME     DEFAULT CURRENT_TIMESTAMP COMMENT '操作时间',
  PRIMARY KEY (id),
  KEY idx_oper_time (oper_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='审计日志表';

-- 17. 短信日志
DROP TABLE IF EXISTS sys_sms_log;
CREATE TABLE sys_sms_log (
  id              BIGINT       NOT NULL AUTO_INCREMENT,
  phone           VARCHAR(20)  NOT NULL,
  template        VARCHAR(50)  DEFAULT NULL,
  content         VARCHAR(500) DEFAULT NULL,
  send_result     VARCHAR(50)  DEFAULT NULL,
  send_time       DATETIME     DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_send_time (send_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='短信日志表';

-- 18. 应用授权记录
DROP TABLE IF EXISTS u_grant;
CREATE TABLE u_grant (
  id              BIGINT       NOT NULL AUTO_INCREMENT,
  user_id         BIGINT       NOT NULL COMMENT '用户ID',
  user_type       VARCHAR(10)  DEFAULT 'user' COMMENT '用户类型: user-自然人 corp-法人',
  app_id          VARCHAR(32)  NOT NULL COMMENT '应用AppId',
  grant_time      DATETIME     DEFAULT CURRENT_TIMESTAMP COMMENT '授权时间',
  expire_time     DATETIME     DEFAULT NULL COMMENT '过期时间',
  status          TINYINT      DEFAULT 1 COMMENT '状态: 0-失效 1-有效',
  PRIMARY KEY (id),
  KEY idx_user_app (user_id, app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='应用授权记录表';

-- ============================================================
-- 四、初始化数据
-- ============================================================

-- 默认管理员 admin/admin123 （BCrypt加密）
INSERT INTO sys_user (username, password, nickname, status) VALUES
('admin', '$2a$10$OWqmV9735md2JIra29db9.aTlaWa3ES/QVnESelz/AyUX6eBSDsGq', '超级管理员', 1);

-- 默认角色
INSERT INTO sys_role (role_name, role_key, role_sort, status, remark) VALUES
('超级管理员', 'admin', 1, 1, '超级管理员'),
('普通管理员', 'common', 2, 1, '普通管理员');

-- 用户-角色关联
INSERT INTO sys_user_role (user_id, role_id) VALUES (1, 1);

-- 默认菜单（精简版：仅保留UAS平台实际使用的功能）
INSERT INTO sys_menu (menu_name, parent_id, menu_sort, path, component, menu_type, visible, perms, icon) VALUES
('首页', 0, 1, '/index', 'dashboard/Index', 'C', 1, '', 'dashboard'),
('用户管理', 0, 2, '/uas-user', NULL, 'M', 1, '', 'user'),
('自然人用户', 2, 1, '/uas-user/user', 'uas-user/User', 'C', 1, 'uas:user:list', 'user'),
('法人用户', 2, 2, '/uas-user/corp', 'uas-user/Corp', 'C', 1, 'uas:corp:list', 'tree'),
('审核管理', 2, 3, '/uas-user/audit', 'uas-user/Audit', 'C', 1, 'uas:audit:list', 'audit'),
('系统管理', 0, 3, '/system', NULL, 'M', 1, '', 'system'),
('用户管理', 6, 1, '/system/user', 'system/User', 'C', 1, 'system:user:list', 'user'),
('角色管理', 6, 2, '/system/role', 'system/Role', 'C', 1, 'system:role:list', 'peoples'),
('菜单管理', 6, 3, '/system/menu', 'system/Menu', 'C', 1, 'system:menu:list', 'tree-table'),
('应用接入', 0, 4, '/uas-app', NULL, 'M', 1, '', 'app'),
('应用管理', 10, 1, '/uas-app/app', 'uas-app/App', 'C', 1, 'uas:app:list', 'app'),
('授权管理', 10, 2, '/uas-app/grant', 'uas-app/Grant', 'C', 1, 'uas:grant:list', 'lock'),
('统计分析', 0, 5, '/uas-stat', NULL, 'M', 1, '', 'chart'),
('账户统计', 13, 1, '/uas-stat/account', 'uas-stat/Account', 'C', 1, 'stat:account:list', 'chart'),
('登录统计', 13, 2, '/uas-stat/login', 'uas-stat/Login', 'C', 1, 'stat:login:list', 'logininfor'),
('接口统计', 13, 3, '/uas-stat/api', 'uas-stat/Api', 'C', 1, 'stat:api:list', 'monitor'),
('消息统计', 13, 4, '/uas-stat/sms', 'uas-stat/Sms', 'C', 1, 'stat:sms:list', 'message'),
('日志管理', 0, 6, '/log', NULL, 'M', 1, '', 'log'),
('登录日志', 23, 1, '/log/loginLog', 'log/LoginLog', 'C', 1, 'log:login:list', 'logininfor'),
('审计日志', 23, 2, '/log/auditLog', 'log/AuditLog', 'C', 1, 'log:audit:list', 'form'),
('短信日志', 23, 3, '/log/smsLog', 'log/SmsLog', 'C', 1, 'log:sms:list', 'message');

-- 角色-菜单关联（admin角色拥有全部菜单）
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT 1, id FROM sys_menu;

-- 默认部门
INSERT INTO sys_dept (parent_id, dept_name, dept_sort, leader, status) VALUES
(0, '若依科技', 0, '若依', 1),
(1, '深圳总公司', 1, '若依', 1),
(1, '长沙分公司', 2, '若依', 1);

-- 默认岗位
INSERT INTO sys_post (post_code, post_name, post_sort, status, remark) VALUES
('ceo', '董事长', 1, 1, ''),
('se', '项目经理', 2, 1, ''),
('hr', '人力资源', 3, 1, ''),
('user', '普通员工', 4, 1, '');

-- 默认字典类型
INSERT INTO sys_dict_type (dict_name, dict_type, status, remark) VALUES
('用户性别', 'sys_user_sex', 1, '用户性别列表'),
('菜单状态', 'sys_show_hide', 1, '菜单状态列表'),
('系统开关', 'sys_normal_disable', 1, '系统开关列表'),
('任务状态', 'sys_job_status', 1, '任务状态列表'),
('任务分组', 'sys_job_group', 1, '任务分组列表'),
('用户认证等级', 'uas_auth_level', 1, '用户认证等级'),
('用户状态', 'uas_user_status', 1, 'UAS用户状态'),
('应用类型', 'uas_app_type', 1, '应用类型'),
('登录方式', 'uas_login_type', 1, '登录方式'),
('登录结果', 'uas_login_result', 1, '登录结果');

-- 默认字典数据
INSERT INTO sys_dict_data (dict_type, dict_label, dict_value, dict_sort, list_class, is_default, status) VALUES
('sys_user_sex', '男', '1', 1, 'primary', 0, 1),
('sys_user_sex', '女', '2', 2, 'danger', 0, 1),
('sys_user_sex', '未知', '0', 3, 'info', 1, 1),
('sys_normal_disable', '启用', '1', 1, 'primary', 1, 1),
('sys_normal_disable', '禁用', '0', 2, 'danger', 0, 1),
('uas_auth_level', 'L1-基础认证', 'L1', 1, 'info', 1, 1),
('uas_auth_level', 'L2-实名认证', 'L2', 2, 'primary', 0, 1),
('uas_auth_level', 'L3-人脸认证', 'L3', 3, 'success', 0, 1),
('uas_user_status', '启用', '1', 1, 'success', 1, 1),
('uas_user_status', '禁用', '0', 2, 'danger', 0, 1),
('uas_app_type', 'Web应用程序', 'web', 1, 'primary', 1, 1),
('uas_app_type', '移动应用', 'mobile', 2, 'success', 0, 1),
('uas_login_type', '密码登录', 'password', 1, 'primary', 1, 1),
('uas_login_type', '短信登录', 'sms', 2, 'success', 0, 1),
('uas_login_type', '企业登录', 'corp', 3, 'warning', 0, 1),
('uas_login_result', '成功', '1', 1, 'success', 1, 1),
('uas_login_result', '失败', '0', 2, 'danger', 0, 1);

-- 默认参数
INSERT INTO sys_config (config_name, config_key, config_value, config_type, remark) VALUES
('主框架版本', 'sys.index.version', '1.0.0', 'Y', '系统版本'),
('用户初始密码', 'sys.user.initPassword', '123456', 'Y', '用户初始密码'),
('OAuth2授权码有效期', 'uas.oauth.code.expire', '300', 'Y', 'OAuth2授权码有效期（秒）'),
('OAuth2 Token有效期', 'uas.oauth.token.expire', '604800', 'Y', 'OAuth2 Token有效期（秒，默认7天）'),
('Token有效期', 'sys.token.expire', '86400', 'Y', '管理后台Token有效期（秒，默认24小时）');

-- 默认第三方应用：校园二手交易平台
INSERT INTO u_app (app_id, app_name, app_type, sm4_secret, app_secret, redirect_uri, status, description) VALUES
('KK790SCHOOLTRADE', '校园二手交易平台', 'web', 'ST-SchoolTrade-2026SM4SecretKey', 'a15df289schooltradebf6cfbda', 'http://localhost:8080/oauth/callback', 1, '校园二手交易平台接入UAS统一身份认证');

-- 测试自然人用户
INSERT INTO u_user (phone, password, real_name, id_card_no, auth_level, status, audit_status, nickname) VALUES
('18788904282', '$2a$10$OWqmV9735md2JIra29db9.aTlaWa3ES/QVnESelz/AyUX6eBSDsGq', '文伟', '510123199001011234', 'L1', 1, 1, '文伟'),
('13800000001', '$2a$10$OWqmV9735md2JIra29db9.aTlaWa3ES/QVnESelz/AyUX6eBSDsGq', '小艾', '510123199002022345', 'L2', 1, 2, '小艾'),
('13800000002', '$2a$10$OWqmV9735md2JIra29db9.aTlaWa3ES/QVnESelz/AyUX6eBSDsGq', '鲍勃', '510123199003033456', 'L1', 1, 1, '鲍勃');

-- 测试法人用户
INSERT INTO u_corp_user (username, password, corp_name, credit_code, legal_person, phone, status, audit_status) VALUES
('testcorp', '$2a$10$FW61Ao9O7E3e/MPqZ.pi9OfQKf9lZYTefDOycAr6tcAXppQYKFsg6', '测试科技有限公司', '91510123MA6ABCDEF1', '张三', '13800000003', 1, 2);

-- 测试登录日志
INSERT INTO u_login_log (user_id, username, login_type, login_ip, login_result, login_time) VALUES
(1, '18788904282', 'password', '127.0.0.1', 1, NOW()),
(2, '13800000001', 'password', '127.0.0.1', 1, NOW() - INTERVAL 1 HOUR),
(1, 'admin', 'password', '127.0.0.1', 1, NOW() - INTERVAL 2 HOUR);

-- 测试审计日志
INSERT INTO sys_audit_log (oper_name, oper_type, oper_content, oper_ip, oper_time) VALUES
('admin', '登录', '管理员登录系统', '127.0.0.1', NOW()),
('admin', '查询', '查询用户列表', '127.0.0.1', NOW() - INTERVAL 1 HOUR),
('admin', '新增', '新增应用: 校园二手交易平台', '127.0.0.1', NOW() - INTERVAL 2 HOUR);

-- 测试短信日志
INSERT INTO sys_sms_log (phone, template, content, send_result, send_time) VALUES
('18788904282', 'LOGIN_CODE', '【UAS】您的登录验证码为123456，5分钟内有效', 'success', NOW()),
('13800000001', 'LOGIN_CODE', '【UAS】您的登录验证码为654321，5分钟内有效', 'success', NOW() - INTERVAL 1 HOUR);

-- 测试授权记录
INSERT INTO u_grant (user_id, user_type, app_id, grant_time, expire_time, status) VALUES
(1, 'user', 'KK790SCHOOLTRADE', NOW(), DATE_ADD(NOW(), INTERVAL 365 DAY), 1),
(2, 'user', 'KK790SCHOOLTRADE', NOW(), DATE_ADD(NOW(), INTERVAL 365 DAY), 1);
