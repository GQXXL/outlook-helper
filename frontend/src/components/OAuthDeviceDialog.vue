<template>
  <el-dialog
    v-model="visible"
    :title="dialogTitle"
    width="min(640px, calc(100vw - 24px))"
    :before-close="handleClose"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="108px" class="oauth-form">
      <el-form-item label="邮箱地址" prop="email_address">
        <el-input
          v-model="form.email_address"
          placeholder="请输入 Outlook 邮箱"
          :disabled="isReauthorize"
          clearable
        />
      </el-form-item>

      <el-form-item label="客户端ID" prop="client_id">
        <el-input v-model="form.client_id" placeholder="请输入 Microsoft Client ID" clearable />
      </el-form-item>

      <el-form-item v-if="!isReauthorize" label="保存密码">
        <el-input
          v-model="form.password"
          type="password"
          placeholder="可留空，系统保存为 oauth"
          show-password
          clearable
        />
      </el-form-item>

      <el-form-item v-if="!isReauthorize" label="备注">
        <el-input v-model="form.remark" placeholder="请输入备注信息（可选）" clearable />
      </el-form-item>
    </el-form>

    <section v-if="deviceCode" class="oauth-panel">
      <div class="auth-code-row">
        <span class="auth-code-label">授权码</span>
        <strong class="auth-code">{{ deviceCode.user_code }}</strong>
        <el-button circle size="small" @click="copyText(deviceCode.user_code)" title="复制授权码">
          <el-icon><CopyDocument /></el-icon>
        </el-button>
      </div>

      <a class="auth-link" :href="deviceCode.verification_uri" target="_blank" rel="noopener">
        <el-icon><Link /></el-icon>
        {{ deviceCode.verification_uri }}
      </a>

      <div class="auth-status-row">
        <el-tag :type="statusTagType" effect="light">{{ statusText }}</el-tag>
        <span v-if="expiresText" class="auth-expiry">{{ expiresText }}</span>
      </div>
    </section>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button v-if="deviceCode" :loading="polling" @click="pollDeviceToken">
          <el-icon><Refresh /></el-icon>
          检测授权
        </el-button>
        <el-button type="primary" :loading="starting || adding" @click="startDeviceAuthorization">
          <el-icon><Key /></el-icon>
          {{ deviceCode ? '重新获取授权码' : '获取授权码' }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { CopyDocument, Key, Link, Refresh } from '@element-plus/icons-vue'
import {
  emailAPI,
  oauthAPI,
  type AddEmailRequest,
  type Email,
  type OAuthDeviceCodeResponse,
} from '@/api'

const DEFAULT_CLIENT_ID = 'd3590ed6-52b3-4102-aeff-aad2292ab01c'

interface Props {
  modelValue: boolean
  mode?: 'add' | 'reauthorize'
  email?: Email | null
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'success'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const visible = ref(false)
const starting = ref(false)
const polling = ref(false)
const adding = ref(false)
const formRef = ref<FormInstance>()
const deviceCode = ref<OAuthDeviceCodeResponse | null>(null)
const expiresAt = ref<number | null>(null)
const status = ref<'idle' | 'pending' | 'authorized' | 'failed'>('idle')
const statusMessage = ref('')
let pollTimer: ReturnType<typeof window.setInterval> | undefined

const form = reactive({
  email_address: '',
  client_id: '',
  password: '',
  remark: '',
})

const isReauthorize = computed(() => props.mode === 'reauthorize' && !!props.email)

const dialogTitle = computed(() => (isReauthorize.value ? '重新授权邮箱' : '授权添加邮箱'))

const actionFailedMessage = computed(() => (isReauthorize.value ? '重新授权失败' : '添加邮箱失败'))

const rules: FormRules = {
  email_address: [
    { required: true, message: '请输入邮箱地址', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' },
  ],
  client_id: [{ required: true, message: '请输入客户端ID', trigger: 'blur' }],
}

const statusText = computed(() => {
  if (adding.value) {
    return isReauthorize.value ? '正在更新授权' : '正在添加邮箱'
  }
  if (status.value === 'authorized') {
    return '授权成功'
  }
  if (status.value === 'failed') {
    return statusMessage.value || '授权失败'
  }
  if (status.value === 'pending') {
    return statusMessage.value || '等待授权'
  }
  return '未开始'
})

const statusTagType = computed(() => {
  if (status.value === 'authorized') {
    return 'success'
  }
  if (status.value === 'failed') {
    return 'danger'
  }
  return 'info'
})

const expiresText = computed(() => {
  if (!deviceCode.value || status.value !== 'pending') {
    return ''
  }

  const minutes = Math.ceil(deviceCode.value.expires_in / 60)
  return `授权码 ${minutes} 分钟内有效`
})

watch(
  () => props.modelValue,
  (val) => {
    visible.value = val
    if (val) {
      resetState()
    }
  },
)

watch(visible, (val) => {
  emit('update:modelValue', val)
})

const clearPollTimer = () => {
  if (pollTimer) {
    window.clearInterval(pollTimer)
    pollTimer = undefined
  }
}

const resetState = () => {
  clearPollTimer()
  if (formRef.value) {
    formRef.value.resetFields()
  }
  Object.assign(form, {
    email_address: props.email?.email_address || '',
    client_id: props.email?.client_id || DEFAULT_CLIENT_ID,
    password: '',
    remark: props.email?.remark || '',
  })
  deviceCode.value = null
  expiresAt.value = null
  status.value = 'idle'
  statusMessage.value = ''
  polling.value = false
  starting.value = false
  adding.value = false
}

const handleClose = () => {
  clearPollTimer()
  visible.value = false
}

const startDeviceAuthorization = async () => {
  if (!formRef.value) return

  try {
    const valid = await formRef.value.validate()
    if (!valid) return
  } catch {
    return
  }

  try {
    clearPollTimer()
    starting.value = true
    const response = await oauthAPI.createDeviceCode({
      client_id: form.client_id.trim(),
    })

    if (!response.data.success || !response.data.data) {
      ElMessage.error(response.data.error || response.data.message || '获取授权码失败')
      return
    }

    deviceCode.value = response.data.data
    expiresAt.value = Date.now() + response.data.data.expires_in * 1000
    status.value = 'pending'
    statusMessage.value = '等待微软账号授权'
    await copyText(response.data.data.user_code, false)
    window.open(response.data.data.verification_uri, '_blank', 'noopener')
    startPolling(response.data.data.interval)
  } catch (error: any) {
    ElMessage.error(
      error.response?.data?.error || error.response?.data?.message || '获取授权码失败',
    )
  } finally {
    starting.value = false
  }
}

const startPolling = (interval: number) => {
  clearPollTimer()
  const delay = Math.max(interval || 5, 3) * 1000
  pollTimer = window.setInterval(() => {
    void pollDeviceToken()
  }, delay)
}

const pollDeviceToken = async () => {
  if (!deviceCode.value || polling.value || adding.value) return

  if (expiresAt.value && Date.now() >= expiresAt.value) {
    status.value = 'failed'
    statusMessage.value = '授权码已过期'
    clearPollTimer()
    return
  }

  try {
    polling.value = true
    const response = await oauthAPI.pollDeviceToken({
      client_id: form.client_id.trim(),
      device_code: deviceCode.value.device_code,
    })

    const result = response.data.data
    if (!response.data.success || !result) {
      status.value = 'failed'
      statusMessage.value = response.data.error || response.data.message || '授权失败'
      clearPollTimer()
      return
    }

    if (result.status === 'pending') {
      status.value = 'pending'
      statusMessage.value =
        result.error === 'slow_down' ? '检测太频繁，稍后继续' : '等待微软账号授权'
      return
    }

    if (result.status === 'failed') {
      status.value = 'failed'
      statusMessage.value = result.error_description || result.error || '授权失败'
      clearPollTimer()
      return
    }

    if (result.status === 'authorized' && result.refresh_token) {
      status.value = 'authorized'
      clearPollTimer()
      await addAuthorizedEmail(result.refresh_token)
    }
  } catch (error: any) {
    status.value = 'failed'
    statusMessage.value = error.response?.data?.error || error.response?.data?.message || '授权失败'
    clearPollTimer()
  } finally {
    polling.value = false
  }
}

const addAuthorizedEmail = async (refreshToken: string) => {
  try {
    adding.value = true
    const response = isReauthorize.value
      ? await emailAPI.reauthorizeEmail(props.email!.id, {
          client_id: form.client_id.trim(),
          refresh_token: refreshToken,
        })
      : await emailAPI.addEmail({
          email_address: form.email_address.trim(),
          password: form.password.trim() || 'oauth',
          client_id: form.client_id.trim(),
          refresh_token: refreshToken,
          remark: form.remark.trim(),
        } as AddEmailRequest)

    if (!response.data.success) {
      status.value = 'failed'
      statusMessage.value =
        response.data.error || response.data.message || actionFailedMessage.value
      ElMessage.error(statusMessage.value)
      return
    }

    ElMessage.success(isReauthorize.value ? '重新授权成功' : '授权成功，邮箱已添加')
    emit('success')
    handleClose()
  } catch (error: any) {
    status.value = 'failed'
    statusMessage.value =
      error.response?.data?.error || error.response?.data?.message || actionFailedMessage.value
    ElMessage.error(statusMessage.value)
  } finally {
    adding.value = false
  }
}

const copyText = async (text: string, showMessage = true) => {
  try {
    await navigator.clipboard.writeText(text)
    if (showMessage) {
      ElMessage.success('已复制')
    }
  } catch {
    const textArea = document.createElement('textarea')
    textArea.value = text
    document.body.appendChild(textArea)
    textArea.select()
    document.execCommand('copy')
    document.body.removeChild(textArea)
    if (showMessage) {
      ElMessage.success('已复制')
    }
  }
}
</script>

<style scoped>
.oauth-form {
  margin-top: 4px;
}

.oauth-panel {
  margin-top: 16px;
  padding: 14px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #ffffff;
}

.auth-code-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.auth-code-label {
  color: #718096;
  font-size: 13px;
}

.auth-code {
  flex: 1;
  min-width: 0;
  color: #111827;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 22px;
  letter-spacing: 0;
  overflow-wrap: anywhere;
}

.auth-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  margin-top: 12px;
  color: #2563eb;
  font-size: 14px;
  text-decoration: none;
  overflow-wrap: anywhere;
}

.auth-link:hover {
  text-decoration: underline;
}

.auth-status-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

.auth-expiry {
  color: #718096;
  font-size: 13px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.dialog-footer :deep(.el-button) {
  margin-left: 0;
}

@media (max-width: 768px) {
  :deep(.el-form-item) {
    display: block;
  }

  :deep(.el-form-item__label) {
    width: auto !important;
    justify-content: flex-start;
    margin-bottom: 6px;
  }

  :deep(.el-form-item__content) {
    margin-left: 0 !important;
  }

  .dialog-footer {
    display: grid;
    grid-template-columns: 1fr;
  }

  .dialog-footer :deep(.el-button) {
    width: 100%;
  }
}
</style>
