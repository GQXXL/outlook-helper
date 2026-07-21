<template>
  <div class="access-codes-page">
    <el-card class="operation-card">
      <div class="operation-bar">
        <div>
          <h3>授权码管理</h3>
          <p>给用户分配独立授权码，并绑定允许查看的邮箱。</p>
        </div>
        <div class="operation-actions">
          <el-button @click="loadData">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
          <el-button type="primary" @click="openCreateDialog">
            <el-icon><Plus /></el-icon>
            新增授权码
          </el-button>
        </div>
      </div>
    </el-card>

    <el-card class="table-card">
      <el-table v-loading="loading" :data="accessCodes" style="width: 100%">
        <el-table-column prop="name" label="名称" min-width="180" />
        <el-table-column prop="email_count" label="绑定邮箱数" width="120" />
        <el-table-column prop="enabled" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'">
              {{ row.enabled ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button link type="warning" @click="rotateAccessCode(row)">重置授权码</el-button>
            <el-button link type="danger" @click="deleteAccessCode(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="editingAccessCode ? '编辑授权码' : '新增授权码'"
      width="640px"
    >
      <el-form label-width="90px">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="例如：用户A、客户B" />
        </el-form-item>
        <el-form-item v-if="editingAccessCode" label="状态">
          <el-switch v-model="form.enabled" active-text="启用" inactive-text="停用" />
        </el-form-item>
        <el-form-item label="可看邮箱">
          <div class="email-selector">
            <el-checkbox-group v-model="form.email_ids">
              <el-checkbox
                v-for="email in emails"
                :key="email.id"
                :label="email.id"
              >
                {{ email.email_address }}
                <span v-if="email.remark" class="email-remark">({{ email.remark }})</span>
              </el-checkbox>
            </el-checkbox-group>
            <el-empty v-if="emails.length === 0" description="暂无邮箱，请先添加邮箱" />
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveAccessCode">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="codeDialogVisible" title="请立即保存授权码" width="560px">
      <el-alert
        title="授权码明文只会展示这一次，关闭后无法再次查看，只能重置。"
        type="warning"
        :closable="false"
        show-icon
      />
      <el-input v-model="generatedCode" readonly class="generated-code">
        <template #append>
          <el-button @click="copyGeneratedCode">
            <el-icon><CopyDocument /></el-icon>
          </el-button>
        </template>
      </el-input>
      <template #footer>
        <el-button type="primary" @click="codeDialogVisible = false">我已保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CopyDocument, Plus, Refresh } from '@element-plus/icons-vue'
import {
  accessCodeAPI,
  emailAPI,
  type AccessCode,
  type Email
} from '@/api'

const loading = ref(false)
const saving = ref(false)
const accessCodes = ref<AccessCode[]>([])
const emails = ref<Email[]>([])
const dialogVisible = ref(false)
const codeDialogVisible = ref(false)
const generatedCode = ref('')
const editingAccessCode = ref<AccessCode | null>(null)

const form = reactive({
  name: '',
  enabled: true,
  email_ids: [] as number[]
})

const loadData = async () => {
  loading.value = true
  try {
    const [accessCodeRes, allEmails] = await Promise.all([
      accessCodeAPI.getAccessCodes(),
      loadAllEmails()
    ])

    if (accessCodeRes.data.success) {
      accessCodes.value = accessCodeRes.data.data || []
    }

    emails.value = allEmails
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '加载授权码数据失败')
  } finally {
    loading.value = false
  }
}

const loadAllEmails = async () => {
  const pageSize = 100
  let offset = 0
  let total = 0
  const result: Email[] = []

  do {
    const response = await emailAPI.getEmails({ limit: pageSize, offset })
    if (!response.data.success) {
      break
    }

    const data = response.data.data as any
    const list = data.list || []
    total = data.total || list.length
    result.push(...list)
    offset += pageSize
  } while (result.length < total)

  return result
}

const resetForm = () => {
  form.name = ''
  form.enabled = true
  form.email_ids = []
}

const openCreateDialog = () => {
  editingAccessCode.value = null
  resetForm()
  dialogVisible.value = true
}

const openEditDialog = (accessCode: AccessCode) => {
  editingAccessCode.value = accessCode
  form.name = accessCode.name
  form.enabled = accessCode.enabled
  form.email_ids = [...(accessCode.email_ids || [])]
  dialogVisible.value = true
}

const saveAccessCode = async () => {
  if (!form.name.trim()) {
    ElMessage.warning('请输入授权码名称')
    return
  }

  saving.value = true
  try {
    if (editingAccessCode.value) {
      await accessCodeAPI.updateAccessCode(editingAccessCode.value.id, {
        name: form.name.trim(),
        enabled: form.enabled,
        email_ids: form.email_ids
      })
      ElMessage.success('授权码已更新')
    } else {
      const response = await accessCodeAPI.createAccessCode({
        name: form.name.trim(),
        email_ids: form.email_ids
      })
      if (response.data.data?.code) {
        generatedCode.value = response.data.data.code
        codeDialogVisible.value = true
      }
      ElMessage.success('授权码已创建')
    }

    dialogVisible.value = false
    loadData()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '保存授权码失败')
  } finally {
    saving.value = false
  }
}

const rotateAccessCode = async (accessCode: AccessCode) => {
  try {
    await ElMessageBox.confirm(
      `确定要重置 ${accessCode.name} 的授权码吗？旧授权码会立即失效。`,
      '重置授权码',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    const response = await accessCodeAPI.rotateAccessCode(accessCode.id)
    if (response.data.data?.code) {
      generatedCode.value = response.data.data.code
      codeDialogVisible.value = true
    }
    ElMessage.success('授权码已重置')
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '重置授权码失败')
    }
  }
}

const deleteAccessCode = async (accessCode: AccessCode) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除 ${accessCode.name} 吗？`,
      '删除授权码',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await accessCodeAPI.deleteAccessCode(accessCode.id)
    ElMessage.success('授权码已删除')
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '删除授权码失败')
    }
  }
}

const copyGeneratedCode = async () => {
  try {
    await navigator.clipboard.writeText(generatedCode.value)
    ElMessage.success('授权码已复制')
  } catch (error) {
    ElMessage.warning('复制失败，请手动复制')
  }
}

const formatTime = (timeStr: string): string => {
  return new Date(timeStr).toLocaleString('zh-CN')
}

onMounted(loadData)
</script>

<style scoped>
.access-codes-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.operation-card,
.table-card {
  border-radius: 8px;
}

.operation-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.operation-bar h3 {
  margin: 0 0 4px;
  color: #2d3748;
}

.operation-bar p {
  margin: 0;
  color: #718096;
}

.operation-actions {
  display: flex;
  gap: 8px;
}

.email-selector {
  max-height: 260px;
  overflow: auto;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 8px 12px;
  width: 100%;
}

.email-selector :deep(.el-checkbox) {
  display: flex;
  margin-right: 0;
  min-height: 30px;
}

.email-remark {
  color: #718096;
  margin-left: 4px;
}

.generated-code {
  margin-top: 16px;
}
</style>
