<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="brand">
        <div class="brand-logo">V</div>
        <h1 class="brand-name">加入 VIBE</h1>
        <p class="brand-slogan">注册账号，开始井井有条的生活</p>
      </div>

      <el-form ref="formRef" :model="form" :rules="rules" size="large">
        <el-form-item prop="username">
          <el-input v-model="form.username" placeholder="用户名（2~32 位字母/数字/下划线）" :prefix-icon="User" />
        </el-form-item>
        <el-form-item prop="nickname">
          <el-input v-model="form.nickname" placeholder="昵称" :prefix-icon="Postcard" />
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="form.password" type="password" placeholder="密码（至少 8 位，含字母和数字）" show-password :prefix-icon="Lock" />
        </el-form-item>
        <el-form-item prop="confirm">
          <el-input v-model="form.confirm" type="password" placeholder="确认密码" show-password :prefix-icon="Lock" />
        </el-form-item>
        <el-button type="primary" class="submit" round :loading="loading" @click="submit">注 册</el-button>
      </el-form>

      <div class="auth-footer">
        已有账号？
        <router-link class="link" :to="{ name: 'login' }">直接登录</router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { Lock, Postcard, User } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()
const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({ username: '', nickname: '', password: '', confirm: '' })

const rules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_]{2,32}$/, message: '2~32 位字母、数字或下划线', trigger: 'blur' },
  ],
  nickname: [{ required: true, message: '请输入昵称', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, message: '密码至少 8 位', trigger: 'blur' },
    {
      validator: (_r, v, cb) => {
        if (v && !/[a-zA-Z]/.test(v)) return cb(new Error('密码需包含字母'))
        if (v && !/\d/.test(v)) return cb(new Error('密码需包含数字'))
        cb()
      },
      trigger: 'blur',
    },
  ],
  confirm: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    {
      validator: (_r, v, cb) => (v === form.password ? cb() : cb(new Error('两次输入的密码不一致'))),
      trigger: 'blur',
    },
  ],
}

async function submit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    await authStore.register(form.username, form.password, form.nickname)
    ElMessage.success('注册成功，请登录')
    router.replace({ name: 'login' })
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page {
  height: 100%;
  display: grid;
  place-items: center;
  background: radial-gradient(1200px 700px at 80% 15%, #f3f0ff 0%, var(--vibe-bg) 55%);
}
.auth-card {
  width: 400px;
  padding: 40px 40px 28px;
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
  margin-bottom: 24px;
}
.brand-logo {
  width: 52px;
  height: 52px;
  margin: 0 auto 10px;
  display: grid;
  place-items: center;
  border-radius: 16px;
  background: linear-gradient(135deg, #7c8cff, #5a69e6);
  color: #fff;
  font-size: 26px;
  font-weight: 800;
  box-shadow: 0 12px 28px rgba(108, 123, 255, 0.4);
}
.brand-name {
  margin: 0;
  font-size: 20px;
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
  margin-top: 20px;
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
