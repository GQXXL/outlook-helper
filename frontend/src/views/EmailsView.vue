<template>
  <div class="emails-page">
    <!-- 操作栏 -->
    <el-card class="operation-card">
      <div class="operation-bar">
        <div class="operation-left">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索邮箱地址或备注"
            class="search-input"
            clearable
            @input="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <div class="mailbox-switch">
            <span class="mailbox-label">邮箱位置</span>
            <el-segmented
              v-model="selectedMailbox"
              :options="mailboxOptions"
              size="default"
            />
          </div>
        </div>
        <div class="operation-right">
          <template v-if="authStore.isAdmin">
          <el-button
            type="danger"
            :disabled="selectedEmails.length === 0 || operationLoading.batchClearInbox"
            :loading="operationLoading.batchClearInbox"
            @click="batchClearInbox"
          >
            <el-icon><Delete /></el-icon>
            清空收件箱
          </el-button>
          <el-button
            type="danger"
            plain
            :disabled="selectedEmails.length === 0 || operationLoading.batchDelete"
            :loading="operationLoading.batchDelete"
            @click="batchDelete"
          >
            <el-icon><Delete /></el-icon>
            批量删除
          </el-button>
          <el-button type="primary" @click="showAddDialog = true">
            <el-icon><Plus /></el-icon>
            添加邮箱
          </el-button>
          <el-button @click="showBatchDialog = true">
            <el-icon><Upload /></el-icon>
            批量添加
          </el-button>
          <el-button @click="showImportDialog = true">
            <el-icon><Document /></el-icon>
            导入文件
          </el-button>
          <el-button @click="showExportDialog = true">
            <el-icon><Download /></el-icon>
            导出
          </el-button>
          </template>
          <el-button @click="loadEmails">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </div>
    </el-card>

    <!-- 邮箱列表 -->
    <el-card class="table-card" v-loading="pageLoading" :element-loading-text="pageLoadingText">
      <el-table
        v-loading="loading"
        :data="emails"
        class="desktop-email-table"
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column v-if="authStore.isAdmin" type="selection" width="55" />
        <el-table-column prop="email_address" label="邮箱地址" width="320">
          <template #default="{ row }">
            <div class="email-address-cell">
              <span
                class="email-address-link"
                @click="handleEmailClick(row)"
                :title="authStore.isAdmin ? '点击标记邮箱' : '邮箱地址'"
              >
                {{ row.email_address }}
              </span>
              <el-button
                size="small"
                type="text"
                class="copy-button"
                @click.stop="copyEmailAddress(row.email_address)"
                title="复制邮箱地址"
              >
                <el-icon size="12"><CopyDocument /></el-icon>
              </el-button>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="标签" min-width="130">
          <template #default="{ row }">
            <div v-if="row.tags && row.tags.length > 0" class="tags-container">
              <el-tag
                v-for="tag in row.tags"
                :key="tag.id"
                :color="tag.color"
                size="small"
                class="tag-item"
              >
                {{ tag.name }}
              </el-tag>
            </div>
            <span v-else class="no-tags">-</span>
            </template>
          </el-table-column>
        <el-table-column prop="remark" label="备注" width="100">
          <template #default="{ row }">
            <el-tooltip
              :content="row.remark || '无备注'"
              placement="top"
              :disabled="!row.remark || row.remark.length <= 10"
            >
              <span class="remark-text">{{ row.remark || '-' }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
          <el-table-column prop="created_at" label="添加时间" width="160">
            <template #default="{ row }">
              {{ formatTime(row.created_at) }}
            </template>
          </el-table-column>
          <el-table-column prop="last_operation_at" label="最后操作" width="160">
            <template #default="{ row }">
              {{ row.last_operation_at ? formatTime(row.last_operation_at) : '-' }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <div class="operation-buttons">
                <el-button
                  link
                  class="icon-button icon-button-primary"
                  :loading="operationLoading.getLatestMail[row.id]"
                  :disabled="operationLoading.getLatestMail[row.id]"
                  @click="getLatestMail(row)"
                >
                  <el-icon :size="18"><Bell /></el-icon>
                </el-button>

                <el-button
                  link
                  class="icon-button icon-button-info"
                  :loading="operationLoading.getAllMails[row.id]"
                  :disabled="operationLoading.getAllMails[row.id]"
                  @click="getAllMails(row)"
                >
                  <el-icon :size="18"><Grid /></el-icon>
                </el-button>

                <el-button
                  v-if="authStore.isAdmin"
                  link
                  class="icon-button icon-button-danger"
                  :loading="operationLoading.deleteEmail[row.id]"
                  :disabled="operationLoading.deleteEmail[row.id]"
                  @click="deleteEmail(row)"
                >
                  <el-icon :size="18"><Close /></el-icon>
                </el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>

        <div class="mobile-email-list" v-loading="loading">
          <el-empty v-if="!loading && emails.length === 0" description="暂无邮箱" />

          <article
            v-for="email in emails"
            :key="email.id"
            class="mobile-email-card"
            :class="{ selected: isEmailSelected(email) }"
          >
            <div class="mobile-card-header">
              <el-checkbox
                v-if="authStore.isAdmin"
                :model-value="isEmailSelected(email)"
                @change="handleMobileSelectionChange(email, $event)"
              />

              <div class="mobile-email-main">
                <button
                  type="button"
                  class="mobile-email-address"
                  :title="authStore.isAdmin ? '点击标记邮箱' : '邮箱地址'"
                  @click="handleEmailClick(email)"
                >
                  {{ email.email_address }}
                </button>
                <div class="mobile-email-subtitle">
                  {{ email.remark || '无备注' }}
                </div>
              </div>

              <el-button
                size="small"
                circle
                class="mobile-copy-button"
                @click.stop="copyEmailAddress(email.email_address)"
                title="复制邮箱地址"
              >
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>

            <div class="mobile-tags-row">
              <template v-if="email.tags && email.tags.length > 0">
                <el-tag
                  v-for="tag in email.tags"
                  :key="tag.id"
                  :color="tag.color"
                  size="small"
                  class="tag-item"
                >
                  {{ tag.name }}
                </el-tag>
              </template>
              <span v-else class="mobile-empty-tags">暂无标签</span>
            </div>

            <div class="mobile-card-meta">
              <div>
                <span>添加时间</span>
                <strong>{{ formatTime(email.created_at) }}</strong>
              </div>
              <div>
                <span>最后操作</span>
                <strong>{{ email.last_operation_at ? formatTime(email.last_operation_at) : '-' }}</strong>
              </div>
            </div>

            <div class="mobile-card-actions">
              <el-button
                type="primary"
                plain
                :loading="operationLoading.getLatestMail[email.id]"
                :disabled="operationLoading.getLatestMail[email.id]"
                @click="getLatestMail(email)"
              >
                <el-icon><Bell /></el-icon>
                最新邮件
              </el-button>

              <el-button
                plain
                :loading="operationLoading.getAllMails[email.id]"
                :disabled="operationLoading.getAllMails[email.id]"
                @click="getAllMails(email)"
              >
                <el-icon><Grid /></el-icon>
                全部邮件
              </el-button>

              <el-button
                v-if="authStore.isAdmin"
                type="danger"
                plain
                :loading="operationLoading.deleteEmail[email.id]"
                :disabled="operationLoading.deleteEmail[email.id]"
                @click="deleteEmail(email)"
              >
                <el-icon><Close /></el-icon>
                删除
              </el-button>
            </div>
          </article>
        </div>

        <!-- 分页 -->
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </el-card>




    <!-- 添加邮箱对话框 -->
    <AddEmailDialog
      v-if="authStore.isAdmin"
      v-model="showAddDialog"
      @success="handleAddSuccess"
    />

    <!-- 批量添加对话框 -->
    <BatchAddDialog
      v-if="authStore.isAdmin"
      v-model="showBatchDialog"
      @success="handleBatchSuccess"
    />

    <!-- 批量标签对话框 -->
    <BatchTagDialog
      v-if="authStore.isAdmin"
      v-model="showBatchTagDialog"
      :email-ids="selectedEmails.map(e => e.id)"
      @success="handleBatchTagSuccess"
    />

    <!-- 邮件查看对话框 -->
    <MailViewDialog
      v-model="showMailDialog"
      :mail-data="currentMail"
      :mail-list="currentMailList"
    />

    <!-- 文件导入对话框 -->
    <FileImportDialog
      v-if="authStore.isAdmin"
      v-model="showImportDialog"
      @success="handleImportSuccess"
    />

    <!-- 导出对话框 -->
    <ExportDialog
      v-if="authStore.isAdmin"
      v-model="showExportDialog"
      :selected-count="selectedEmails.length"
      :selected-email-ids="selectedEmails.map(e => e.id)"
      @export="handleExport"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus,
  Upload,
  Document,
  Search,
  Refresh,
  Message,
  Folder,
  Files,
  Delete,
  DeleteFilled,
  ArrowDown,
  Collection,
  CopyDocument,
  Promotion,
  FolderOpened,
  Bell,
  Grid,
  Close,
  Download
} from '@element-plus/icons-vue'
import AddEmailDialog from '@/components/AddEmailDialog.vue'
import BatchAddDialog from '@/components/BatchAddDialog.vue'
import BatchTagDialog from '@/components/BatchTagDialog.vue'
import MailViewDialog from '@/components/MailViewDialog.vue'
import FileImportDialog from '@/components/FileImportDialog.vue'
import ExportDialog from '@/components/ExportDialog.vue'
import { emailAPI, type Email, type OutlookMail } from '@/api'
import { useAuthStore } from '@/stores/auth'

// 响应式数据
const authStore = useAuthStore()
const loading = ref(false)
const emails = ref<Email[]>([])
const selectedEmails = ref<Email[]>([])
const searchKeyword = ref('')
const currentPage = ref(1)
const pageSize = ref(10)
const total = ref(0)
const selectedMailbox = ref<'INBOX' | 'Junk'>('INBOX')
const mailboxOptions = [
  { label: '收件箱', value: 'INBOX' },
  { label: '垃圾邮件', value: 'Junk' }
]

// 操作loading状态
const operationLoading = ref({
  getLatestMail: {} as Record<number, boolean>,
  getAllMails: {} as Record<number, boolean>,
  deleteEmail: {} as Record<number, boolean>,
  batchClearInbox: false,
  batchDelete: false
})

// 对话框状态
const showAddDialog = ref(false)
const showBatchDialog = ref(false)
const showBatchTagDialog = ref(false)
const showMailDialog = ref(false)
const showImportDialog = ref(false)
const showExportDialog = ref(false)

// 全页面加载遮罩
const pageLoading = ref(false)
const pageLoadingText = ref('加载中...')

// 邮件相关
const currentMail = ref<OutlookMail | null>(null)
const currentMailList = ref<OutlookMail[]>([])

const mailboxName = () => selectedMailbox.value === 'Junk' ? '垃圾邮件' : '收件箱'

// 方法
const loadEmails = async () => {
  loading.value = true
  try {
    const params = {
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value,
      keyword: searchKeyword.value || undefined
    }
    
    const response = await emailAPI.getEmails(params)
    if (response.data.success) {
      const data = response.data.data as any
      emails.value = data.list || []
      total.value = data.total || 0
    } else {
      ElMessage.error(response.data.message || '获取邮箱列表失败')
    }
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '获取邮箱列表失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  currentPage.value = 1
  loadEmails()
}

const handleSelectionChange = (selection: Email[]) => {
  selectedEmails.value = selection
}

const isEmailSelected = (email: Email) => {
  return selectedEmails.value.some(item => item.id === email.id)
}

const toggleEmailSelection = (email: Email, checked: boolean) => {
  if (checked) {
    if (!isEmailSelected(email)) {
      selectedEmails.value = [...selectedEmails.value, email]
    }
    return
  }

  selectedEmails.value = selectedEmails.value.filter(item => item.id !== email.id)
}

const handleMobileSelectionChange = (email: Email, checked: string | number | boolean) => {
  toggleEmailSelection(email, Boolean(checked))
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
  loadEmails()
}

const handleCurrentChange = (page: number) => {
  currentPage.value = page
  loadEmails()
}

const handleImportSuccess = () => {
  loadEmails()
}

const handleExport = async (params: any) => {
  try {
    const exportParams = {
      range: params.range,
      format: params.format,
      field_order: params.fieldOrder,
      email_ids: params.emailIds
    }

    const response = await emailAPI.exportEmails(exportParams)

    const exportData = response.data.data
    if (response.data.success && exportData) {
      // 创建下载链接
      const blob = new Blob([exportData.content], {
        type: params.format === 'csv' ? 'text/csv' : 'text/plain'
      })
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url

      // 设置文件名
      const timestamp = new Date().toISOString().slice(0, 19).replace(/[:]/g, '-')
      const extension = params.format === 'csv' ? 'csv' : 'txt'
      link.download = `outlook_emails_${timestamp}.${extension}`

      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)

      ElMessage.success(`成功导出 ${exportData.count} 个邮箱`)
    } else {
      ElMessage.error(response.data.message || '导出失败')
    }
  } catch (error: any) {
    console.error('导出失败:', error)
    ElMessage.error(error.response?.data?.message || '导出失败，请重试')
  }
}

const copyEmailAddress = async (emailAddress: string) => {
  try {
    await navigator.clipboard.writeText(emailAddress)
    ElMessage.success('邮箱地址已复制到剪贴板')
  } catch (error) {
    // 降级方案
    const textArea = document.createElement('textarea')
    textArea.value = emailAddress
    document.body.appendChild(textArea)
    textArea.select()
    document.execCommand('copy')
    document.body.removeChild(textArea)
    ElMessage.success('邮箱地址已复制到剪贴板')
  }
}

const getLatestMail = async (email: Email) => {
  operationLoading.value.getLatestMail[email.id] = true
  pageLoading.value = true
  pageLoadingText.value = `正在获取${mailboxName()}最新邮件...`
  try {
    const response = await emailAPI.getLatestMail(email.id, selectedMailbox.value)
    if (response.data.success && response.data.data) {
      currentMail.value = response.data.data
      currentMailList.value = [response.data.data]
      showMailDialog.value = true
    } else {
      // 检查是否是"Nothing to fetch"错误
      if (response.data.error && response.data.error.includes('Nothing to fetch')) {
        ElMessage.info('当前邮箱暂无邮件')
      } else {
        ElMessage.error(response.data.message || '获取邮件失败')
      }
    }
  } catch (error: any) {
    // 检查错误响应中是否包含"Nothing to fetch"
    const errorMessage = error.response?.data?.error || error.response?.data?.message || ''
    if (errorMessage.includes('Nothing to fetch')) {
      ElMessage.info('当前邮箱暂无邮件')
    } else {
      ElMessage.error(error.response?.data?.message || '获取邮件失败')
    }
  } finally {
    operationLoading.value.getLatestMail[email.id] = false
    pageLoading.value = false
  }
}

const getAllMails = async (email: Email) => {
  operationLoading.value.getAllMails[email.id] = true
  pageLoading.value = true
  pageLoadingText.value = `正在获取${mailboxName()}全部邮件...`
  try {
    const response = await emailAPI.getAllMails(email.id, selectedMailbox.value)
    if (response.data.success && response.data.data) {
      currentMailList.value = response.data.data
      currentMail.value = response.data.data[0] || null
      showMailDialog.value = true
    } else {
      // 检查是否是"Nothing to fetch"错误
      if (response.data.error && response.data.error.includes('Nothing to fetch')) {
        ElMessage.info('当前邮箱暂无邮件')
      } else {
        ElMessage.error(response.data.message || '获取邮件失败')
      }
    }
  } catch (error: any) {
    // 检查错误响应中是否包含"Nothing to fetch"
    const errorMessage = error.response?.data?.error || error.response?.data?.message || ''
    if (errorMessage.includes('Nothing to fetch')) {
      ElMessage.info('当前邮箱暂无邮件')
    } else {
      ElMessage.error(error.response?.data?.message || '获取邮件失败')
    }
  } finally {
    operationLoading.value.getAllMails[email.id] = false
    pageLoading.value = false
  }
}

const clearInbox = async (email: Email) => {
  try {
    await ElMessageBox.confirm(
      `确定要清空邮箱 ${email.email_address} 的收件箱吗？此操作不可恢复！`,
      '警告',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    const response = await emailAPI.clearInbox(email.id)
    if (response.data.success) {
      ElMessage.success('收件箱清空成功')
      loadEmails()
    } else {
      ElMessage.error(response.data.message || '清空收件箱失败')
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '清空收件箱失败')
    }
  }
}

const batchClearInbox = async () => {
  if (selectedEmails.value.length === 0) {
    ElMessage.warning('请先选择要清空收件箱的邮箱')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要清空选中的 ${selectedEmails.value.length} 个邮箱的收件箱吗？此操作不可恢复！`,
      '警告',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    operationLoading.value.batchClearInbox = true
    const emailIds = selectedEmails.value.map(email => email.id)
    const response = await emailAPI.batchClearInbox(emailIds)

    if (response.data.success) {
      const data = response.data.data as any
      if (data.error_count > 0) {
        ElMessage.warning(`批量清空收件箱完成，成功: ${data.success_count}, 失败: ${data.error_count}`)
      } else {
        ElMessage.success(`成功清空 ${data.success_count} 个邮箱的收件箱`)
      }
      selectedEmails.value = []
      loadEmails()
    } else {
      ElMessage.error(response.data.message || '批量清空收件箱失败')
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '批量清空收件箱失败')
    }
  } finally {
    operationLoading.value.batchClearInbox = false
  }
}

const handleDropdownCommand = (command: string, email: Email) => {
  if (command === 'tag') {
    selectedEmails.value = [email]
    showBatchTagDialog.value = true
  } else if (command === 'delete') {
    deleteEmail(email)
  }
}

const deleteEmail = async (email: Email) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除邮箱 ${email.email_address} 吗？`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    operationLoading.value.deleteEmail[email.id] = true
    pageLoading.value = true
    pageLoadingText.value = '正在删除邮箱...'
    const response = await emailAPI.deleteEmail(email.id)
    if (response.data.success) {
      ElMessage.success('删除成功')
      loadEmails()
    } else {
      ElMessage.error(response.data.message || '删除失败')
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '删除失败')
    }
  } finally {
    operationLoading.value.deleteEmail[email.id] = false
    pageLoading.value = false
  }
}

const batchDelete = async () => {
  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedEmails.value.length} 个邮箱吗？`,
      '确认批量删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    operationLoading.value.batchDelete = true
    // 批量删除
    const emailIds = selectedEmails.value.map(email => email.id)
    await emailAPI.batchDeleteEmails(emailIds)

    ElMessage.success('批量删除成功')
    selectedEmails.value = []
    loadEmails()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '批量删除失败')
    }
  } finally {
    operationLoading.value.batchDelete = false
  }
}

const handleAddSuccess = () => {
  loadEmails()
}

const handleBatchSuccess = () => {
  loadEmails()
}

const handleBatchTagSuccess = () => {
  selectedEmails.value = []
  loadEmails()
}

const handleEmailClick = (email: Email) => {
  if (!authStore.isAdmin) {
    return
  }
  // 设置选中的邮箱为当前点击的邮箱
  selectedEmails.value = [email]
  // 打开批量标签对话框
  showBatchTagDialog.value = true
}

const formatTime = (timeStr: string): string => {
  return new Date(timeStr).toLocaleString('zh-CN')
}

onMounted(() => {
  loadEmails()
})
</script>

<style scoped>
.emails-page {
  width: 100%;
  max-width: 100%;
  margin: 0 auto;
  padding: 0 16px;
  min-width: 0;
}

@media (min-width: 1200px) {
  .emails-page {
    max-width: 1800px;
  }
}

.operation-card {
  margin-bottom: 20px;
}

.operation-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
}

.operation-left {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  align-items: center;
  min-width: 0;
}

.search-input {
  width: 300px;
}

.mailbox-switch {
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
}

.mailbox-label {
  color: #4a5568;
  font-size: 14px;
}

.operation-right {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.table-card {
  margin-bottom: 20px;
  overflow: hidden;
}

.mobile-email-list {
  display: none;
}

.tags-container {
  display: flex;
  flex-wrap: nowrap;
  gap: 2px;
  overflow: hidden;
  align-items: center;
}

.tag-item {
  color: white;
  border: none;
  height: 22px;
  line-height: 20px;
  padding: 0 7px;
  font-size: 12px;
}

.tags-container .el-tag {
  max-width: 90px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  height: 22px;
  line-height: 20px;
  padding: 0 7px;
  font-size: 12px;
}

.no-tags {
  color: #999;
}



:deep(.el-table .cell) {
  padding: 4px 8px;
  font-size: 14px;
}

:deep(.el-table td) {
  height: 36px;
}

:deep(.el-table th) {
  height: 40px;
  padding: 8px 0;
}

:deep(.el-table .el-table__row) {
  height: 36px;
}

.table-card :deep(.el-card__body) {
  overflow-x: auto;
}

:deep(.el-pagination) {
  flex-wrap: wrap;
  gap: 8px;
}

:deep(.el-button-group .el-button) {
  margin: 0;
}

.operation-buttons {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: nowrap;
  height: 28px;
}

.operation-buttons .el-button {
  margin: 0;
}

.operation-buttons .icon-button {
  padding: 3px;
  border-radius: 4px;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  background: transparent;
  position: relative;
  border: 1px solid transparent;
  height: 26px;
  width: 26px;
}

.operation-buttons .icon-button:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
}

.operation-buttons .icon-button:active {
  transform: translateY(0);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

/* 主要操作按钮 - 最新邮件（紫色） */
.operation-buttons .icon-button-primary .el-icon {
  color: #9c27b0;
}

.operation-buttons .icon-button-primary:hover {
  background: rgba(156, 39, 176, 0.1);
  border-color: rgba(156, 39, 176, 0.2);
}

.operation-buttons .icon-button-primary:hover .el-icon {
  color: #7b1fa2;
}

/* 信息操作按钮 - 全部邮件（黑色） */
.operation-buttons .icon-button-info .el-icon {
  color: #303133;
}

.operation-buttons .icon-button-info:hover {
  background: rgba(48, 49, 51, 0.08);
  border-color: rgba(48, 49, 51, 0.15);
}

.operation-buttons .icon-button-info:hover .el-icon {
  color: #000000;
}

/* 危险操作按钮 - 删除 */
.operation-buttons .icon-button-danger .el-icon {
  color: #f56c6c;
}

.operation-buttons .icon-button-danger:hover {
  background: rgba(245, 108, 108, 0.1);
  border-color: rgba(245, 108, 108, 0.2);
}

.operation-buttons .icon-button-danger:hover .el-icon {
  color: #f23c3c;
}

/* 禁用状态 */
.operation-buttons .icon-button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.operation-buttons .icon-button:disabled:hover {
  background: transparent;
  transform: none;
  box-shadow: none;
  border-color: transparent;
}

.email-address-cell {
  display: flex;
  align-items: center;
  width: 100%;
  white-space: nowrap;
  overflow: hidden;
  height: 28px;
}

.email-address-link {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: pointer;
  color: #409eff;
  transition: all 0.2s ease;
  padding: 2px 4px;
  border-radius: 2px;
  font-size: 14px;
  line-height: 20px;
}

.email-address-link:hover {
  background-color: #f0f9ff;
  color: #66b1ff;
  text-decoration: underline;
}

.email-address-link:active {
  background-color: #e6f7ff;
  color: #409eff;
}

.copy-button {
  margin-left: 4px;
  padding: 1px;
  opacity: 0.6;
  transition: all 0.2s ease;
  border-radius: 2px;
  background: transparent;
  min-height: auto;
  height: 16px;
  width: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  vertical-align: middle;
  flex-shrink: 0;
}

.copy-button:hover {
  opacity: 1;
  background: #f0f9ff;
  color: #409eff;
  transform: scale(1.05);
}

/* 确保复制按钮不会影响表格行的点击 */
.copy-button:focus {
  outline: none;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.2);
}

.remark-text {
  display: inline-block;
  width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mobile-email-card {
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  background: #ffffff;
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.06);
  padding: 14px;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
}

.mobile-email-card.selected {
  border-color: #409eff;
  box-shadow: 0 8px 22px rgba(64, 158, 255, 0.16);
}

.mobile-card-header {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  min-width: 0;
}

.mobile-email-main {
  flex: 1;
  min-width: 0;
}

.mobile-email-address {
  display: block;
  width: 100%;
  padding: 0;
  border: 0;
  background: transparent;
  color: #1f2937;
  font-size: 15px;
  font-weight: 650;
  line-height: 1.35;
  text-align: left;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mobile-email-address:active {
  color: #409eff;
}

.mobile-email-subtitle {
  margin-top: 4px;
  color: #718096;
  font-size: 12px;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mobile-copy-button {
  flex-shrink: 0;
}

.mobile-tags-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  min-height: 24px;
}

.mobile-empty-tags {
  color: #a0aec0;
  font-size: 12px;
}

.mobile-card-meta {
  display: grid;
  grid-template-columns: 1fr;
  gap: 8px;
  margin-top: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  background: #f8fafc;
}

.mobile-card-meta div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-width: 0;
}

.mobile-card-meta span {
  color: #718096;
  font-size: 12px;
  white-space: nowrap;
}

.mobile-card-meta strong {
  color: #2d3748;
  font-size: 12px;
  font-weight: 600;
  text-align: right;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mobile-card-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: 12px;
}

.mobile-card-actions :deep(.el-button) {
  width: 100%;
  margin: 0;
  min-height: 38px;
}

/* 自定义加载遮罩样式 - 仅限于卡片区域 */
.table-card :deep(.el-loading-mask) {
  background-color: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(3px);
  transition: opacity 0.3s ease;
  border-radius: 4px;
}

.table-card :deep(.el-loading-spinner) {
  top: 50%;
  transform: translateY(-50%);
}

.table-card :deep(.el-loading-spinner .el-loading-text) {
  color: #409eff;
  font-size: 14px;
  margin-top: 10px;
  font-weight: 500;
}

.table-card :deep(.el-loading-spinner .circular) {
  width: 42px;
  height: 42px;
  animation: loading-rotate 2s linear infinite;
}

@keyframes loading-rotate {
  100% {
    transform: rotate(360deg);
  }
}

@media (max-width: 768px) {
  .emails-page {
    padding: 0;
  }

  .operation-card {
    margin-bottom: 12px;
  }

  .operation-card :deep(.el-card__body),
  .table-card :deep(.el-card__body) {
    padding: 12px;
  }

  .table-card :deep(.el-card__body) {
    overflow-x: visible;
  }

  .operation-bar {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .operation-left {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
  }

  .search-input {
    width: 100%;
  }

  .operation-right {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
    justify-content: stretch;
  }

  .operation-right :deep(.el-button) {
    width: 100%;
    margin: 0;
  }

  .mailbox-switch {
    width: 100%;
    justify-content: space-between;
  }

  .mailbox-switch :deep(.el-segmented) {
    flex: 1;
  }

  .desktop-email-table {
    display: none;
  }

  .mobile-email-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
    min-height: 160px;
  }

  :deep(.el-pagination) {
    justify-content: flex-start;
    overflow-x: auto;
    padding-bottom: 4px;
  }

  :deep(.el-pagination .el-pagination__sizes),
  :deep(.el-pagination .el-pagination__jump) {
    display: none;
  }
}

@media (max-width: 420px) {
  .operation-right {
    grid-template-columns: 1fr;
  }

  .mobile-card-actions {
    grid-template-columns: 1fr;
  }
}
</style>
