<template>
  <div class="settings">
    <!-- 修改昵称 -->
    <section class="card vibe-card">
      <h2 class="card-title"><el-icon><Postcard /></el-icon>修改昵称</h2>
      <p class="card-desc">昵称会显示在侧边栏与其他位置</p>
      <el-form label-position="top" class="form">
        <el-form-item label="新昵称">
          <el-input v-model="nickname" placeholder="请输入新昵称" maxlength="32" show-word-limit style="max-width: 360px" />
        </el-form-item>
        <el-button type="primary" round :loading="savingNickname" @click="saveNickname">保存昵称</el-button>
      </el-form>
    </section>

    <!-- 修改密码 -->
    <section class="card vibe-card">
      <h2 class="card-title"><el-icon><Lock /></el-icon>修改密码</h2>
      <p class="card-desc">修改成功后所有设备需要重新登录</p>
      <el-form ref="pwdFormRef" :model="pwdForm" :rules="pwdRules" label-position="top" class="form">
        <el-form-item label="原密码" prop="oldPassword">
          <el-input v-model="pwdForm.oldPassword" type="password" show-password placeholder="请输入原密码" style="max-width: 360px" />
        </el-form-item>
        <el-form-item label="新密码" prop="newPassword">
          <el-input
            v-model="pwdForm.newPassword"
            type="password"
            show-password
            placeholder="至少 8 位，需包含字母和数字"
            style="max-width: 360px"
          />
        </el-form-item>
        <el-form-item label="确认新密码" prop="confirmPassword">
          <el-input v-model="pwdForm.confirmPassword" type="password" show-password placeholder="再次输入新密码" style="max-width: 360px" />
        </el-form-item>
        <el-button type="danger" round :loading="savingPassword" @click="savePassword">修改密码</el-button>
      </el-form>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

// ---------- 昵称 ----------
const nickname = ref('')
const savingNickname = ref(false)

onMounted(() => {
  nickname.value = authStore.user?.nickname || ''
})

async function saveNickname() {
  const name = nickname.value.trim()
  if (!name) {
    ElMessage.warning('昵称不能为空')
    return
  }
  savingNickname.value = true
  try {
    await authStore.updateNickname(name)
    ElMessage.success('昵称已更新')
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    savingNickname.value = false
  }
}

// ---------- 密码 ----------
const pwdFormRef = ref<FormInstance>()
const savingPassword = ref(false)
const pwdForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })

const pwdRules: FormRules = {
  oldPassword: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 8, message: '新密码至少 8 位', trigger: 'blur' },
    {
      validator: (_r, v, cb) => {
        if (v && !/[a-zA-Z]/.test(v)) return cb(new Error('新密码需包含字母'))
        if (v && !/\d/.test(v)) return cb(new Error('新密码需包含数字'))
        cb()
      },
      trigger: 'blur',
    },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (_r, v, cb) => (v === pwdForm.newPassword ? cb() : cb(new Error('两次输入的密码不一致'))),
      trigger: 'blur',
    },
  ],
}

async function savePassword() {
  if (!pwdFormRef.value) return
  const valid = await pwdFormRef.value.validate().catch(() => false)
  if (!valid) return
  savingPassword.value = true
  try {
    await authStore.changePassword(pwdForm.oldPassword, pwdForm.newPassword)
    ElMessage.success('密码已修改，请重新登录')
    router.replace({ name: 'login' })
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    savingPassword.value = false
  }
}
</script>

<style scoped>
.settings {
  max-width: 720px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.card {
  padding: 24px 28px;
}
.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 6px;
  font-size: 17px;
  font-weight: 700;
  color: var(--vibe-primary);
}
.card-desc {
  margin: 0 0 18px;
  font-size: 13px;
  color: var(--vibe-text-secondary);
}
.form {
  max-width: 520px;
}
</style>
