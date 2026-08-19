<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="brand">
        <div class="brand-logo">S</div>
        <h1 class="brand-name">SimpleLife</h1>
        <p class="brand-slogan">把生活安排得井井有条</p>
      </div>

      <el-form ref="formRef" :model="form" :rules="rules" size="large" @keyup.enter="submit">
        <el-form-item prop="username">
          <el-input v-model="form.username" placeholder="用户名" :prefix-icon="User" />
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="form.password" type="password" placeholder="密码" show-password :prefix-icon="Lock" />
        </el-form-item>
        <el-button type="primary" class="submit" round :loading="loading" @click="submit">登 录</el-button>
      </el-form>

      <div class="auth-footer">
        还没有账号？
        <router-link class="link" :to="{ name: 'register' }">立即注册</router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { Lock, User } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({ username: '', password: '' })
const rules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function submit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    await authStore.login(form.username, form.password)
    ElMessage.success('欢迎回来')
    requestNotifyPermission()
    const redirect = (route.query.redirect as string) || '/today'
    router.replace(redirect)
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

// 提醒依赖浏览器通知授权：登录后温和请求
function requestNotifyPermission() {
  if ('Notification' in window && Notification.permission === 'default') {
    void Notification.requestPermission()
  }
}
</script>

<style scoped>
.auth-page {
  height: 100%;
  display: grid;
  place-items: center;
  background: radial-gradient(1200px 700px at 20% 10%, #eef0ff 0%, var(--vibe-bg) 55%);
}
.auth-card {
  width: 400px;
  padding: 44px 40px 32px;
  background: var(--vibe-surface);
  border: 1px solid var(--vibe-border);
  border-radius: var(--vibe-radius-xl);
  box-shadow: var(--vibe-shadow-lg);
  animation: card-in 0.5s var(--vibe-ease);
}
@keyframes card-in {
  from {
    opacity: 0;
    transform: translateY(20px) scale(0.98);
  }
}
.brand {
  text-align: center;
  margin-bottom: 30px;
}
.brand-logo {
  width: 56px;
  height: 56px;
  margin: 0 auto 12px;
  display: grid;
  place-items: center;
  border-radius: 18px;
  background: linear-gradient(135deg, #7c8cff, #5a69e6);
  color: #fff;
  font-size: 28px;
  font-weight: 800;
  box-shadow: 0 12px 28px rgba(108, 123, 255, 0.4);
}
.brand-name {
  margin: 0;
  font-size: 24px;
  letter-spacing: 4px;
}
.brand-slogan {
  margin: 6px 0 0;
  font-size: 13px;
  color: var(--vibe-text-secondary);
}
.submit {
  width: 100%;
  height: 44px;
  margin-top: 6px;
  font-size: 15px;
  letter-spacing: 6px;
}
.auth-footer {
  margin-top: 22px;
  text-align: center;
  font-size: 13px;
  color: var(--vibe-text-secondary);
}
.link {
  color: var(--vibe-primary);
  text-decoration: none;
  font-weight: 600;
}
</style>
