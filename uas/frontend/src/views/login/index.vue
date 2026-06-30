<template>
  <div class="login-container">
    <div class="login-bg">
      <div class="bg-shape shape1"></div>
      <div class="bg-shape shape2"></div>
      <div class="bg-shape shape3"></div>
    </div>
    <div class="login-box">
      <div class="login-left">
        <div class="welcome">
          <h1>统一身份认证管理系统</h1>
          <p>Unified Authentication Service</p>
          <p class="desc">统一的身份认证、应用接入、用户管理平台<br/>为校园二手交易平台等第三方应用提供 OAuth2 授权服务</p>
        </div>
      </div>
      <div class="login-right">
        <el-form ref="loginFormRef" :model="loginForm" :rules="rules" class="login-form" @keyup.enter="handleLogin">
          <h2 class="title">用户登录</h2>

          <!-- 用户类型切换 -->
          <el-radio-group v-model="loginForm.userType" class="user-type-tabs" @change="handleTypeChange">
            <el-radio-button label="admin">管理员</el-radio-button>
            <el-radio-button label="personal">个体用户</el-radio-button>
            <el-radio-button label="corp">企业用户</el-radio-button>
          </el-radio-group>

          <el-form-item prop="username">
            <el-input v-model="loginForm.username" :placeholder="usernamePlaceholder" size="large" :prefix-icon="User" />
          </el-form-item>
          <el-form-item prop="password">
            <el-input v-model="loginForm.password" type="password" placeholder="请输入密码" size="large" :prefix-icon="Lock" show-password />
          </el-form-item>
          <el-form-item prop="code">
            <div class="captcha-row">
              <el-input v-model="loginForm.code" placeholder="请输入验证码" size="large" :prefix-icon="Picture" maxlength="6" />
              <img v-if="captcha.img" :src="captcha.img" class="captcha-img" @click="getCaptchaCode" alt="验证码" title="点击刷新" />
              <div v-else class="captcha-img captcha-placeholder" @click="getCaptchaCode">点击加载</div>
            </div>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="large" class="login-btn" :loading="loading" @click="handleLogin">登 录</el-button>
          </el-form-item>
          <div class="tips">
            <template v-if="loginForm.userType === 'admin'">
            <span></span>
          </template>
            <template v-else-if="loginForm.userType === 'personal'">
              <span>个体用户用手机号登录</span>
              <el-link type="primary" :underline="false" @click="$router.push('/register')">立即注册</el-link>
            </template>
            <template v-else>
              <span>企业用户：testcorp / corp123456</span>
            </template>
          </div>
        </el-form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock, Picture } from '@element-plus/icons-vue'
import { login, getCaptcha } from '@/api/auth'

const router = useRouter()
const route = useRoute()
const loginFormRef = ref(null)
const loading = ref(false)

const loginForm = reactive({
  username: 'admin',
  password: 'admin123',
  code: '',
  uuid: '',
  userType: 'admin'
})

const rules = {
  username: [{ required: true, message: '请输入账号', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  code: [{ required: true, message: '请输入验证码', trigger: 'blur' }]
}

const captcha = reactive({ img: '', uuid: '' })

// 不同用户类型的用户名输入框 placeholder
const usernamePlaceholder = computed(() => {
  if (loginForm.userType === 'personal') return '请输入手机号'
  if (loginForm.userType === 'corp') return '请输入企业用户名'
  return '请输入用户名'
})

const getCaptchaCode = async () => {
  try {
    const res = await getCaptcha()
    captcha.img = res.data.img
    captcha.uuid = res.data.uuid
  } catch (e) {}
}

// 切换用户类型时清空表单并刷新验证码
const handleTypeChange = () => {
  loginForm.username = ''
  loginForm.password = ''
  loginForm.code = ''
  getCaptchaCode()
}

const handleLogin = async () => {
  try {
    await loginFormRef.value.validate()
    loading.value = true
    const res = await login({
      username: loginForm.username,
      password: loginForm.password,
      code: loginForm.code,
      uuid: captcha.uuid,
      userType: loginForm.userType
    })
    localStorage.setItem('uas_token', res.data.token)
    localStorage.setItem('uas_user', JSON.stringify({
      username: res.data.username,
      nickname: res.data.nickname,
      role: res.data.role,
      userType: res.data.userType
    }))
    ElMessage.success('登录成功')
    // 个体/企业用户登录后跳到个人信息页，管理员跳回原页面或首页
    let redirect = route.query.redirect ? decodeURIComponent(route.query.redirect) : ''
    if (!redirect) {
      if (loginForm.userType === 'personal' || loginForm.userType === 'corp') {
        redirect = '/profile'
      } else {
        redirect = '/'
      }
    }
    router.push(redirect)
  } catch (e) {
    // 登录失败刷新验证码
    getCaptchaCode()
    loginForm.code = ''
    if (e.message) ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  getCaptchaCode()
})
</script>

<style lang="scss" scoped>
.login-container {
  position: relative;
  height: 100%;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1e3c72 0%, #2a5298 50%, #409eff 100%);
  overflow: hidden;
}

.login-bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
  .bg-shape {
    position: absolute;
    border-radius: 50%;
    opacity: 0.08;
    background: #fff;
  }
  .shape1 { width: 400px; height: 400px; top: -150px; left: -150px; }
  .shape2 { width: 300px; height: 300px; bottom: -100px; right: -100px; }
  .shape3 { width: 200px; height: 200px; top: 40%; left: 30%; }
}

.login-box {
  position: relative;
  z-index: 10;
  display: flex;
  width: 880px;
  height: 520px;
  background: #fff;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 20px 60px rgba(0,0,0,0.3);
}

.login-left {
  flex: 1;
  background: linear-gradient(135deg, #1e3c72 0%, #2a5298 100%);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
  .welcome {
    text-align: center;
    h1 { font-size: 26px; margin-bottom: 12px; }
    p { font-size: 14px; opacity: 0.85; margin-bottom: 8px; }
    .desc { font-size: 13px; line-height: 1.8; opacity: 0.7; margin-top: 16px; }
  }
}

.login-right {
  width: 400px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
}

.login-form {
  width: 100%;
  .title {
    text-align: center;
    margin-bottom: 18px;
    color: #303133;
    font-size: 22px;
  }
  .user-type-tabs {
    display: flex;
    width: 100%;
    margin-bottom: 18px;
    :deep(.el-radio-button) {
      flex: 1;
      .el-radio-button__inner {
        width: 100%;
      }
    }
  }
  .login-btn {
    width: 100%;
    letter-spacing: 4px;
  }
  .captcha-row {
    display: flex;
    gap: 10px;
    width: 100%;
    .el-input { flex: 1; }
    .captcha-img {
      width: 120px;
      height: 40px;
      cursor: pointer;
      border-radius: 4px;
      border: 1px solid #dcdfe6;
      object-fit: cover;
    }
    .captcha-placeholder {
      display: flex;
      align-items: center;
      justify-content: center;
      color: #909399;
      font-size: 12px;
      background: #f5f7fa;
    }
  }
  .tips {
    text-align: center;
    color: #909399;
    font-size: 12px;
    margin-top: 10px;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
  }
}

@media (max-width: 768px) {
  .login-box { width: 90%; height: auto; flex-direction: column; }
  .login-left { display: none; }
  .login-right { width: 100%; padding: 30px; }
}
</style>
