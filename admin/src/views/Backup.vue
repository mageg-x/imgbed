<template>
  <div class="backup-container">
    <el-card class="mb-4">
      <template #header>
        <div class="flex justify-between items-center">
          <h2 class="text-xl font-bold">{{ $t('backup.title') }}</h2>
          <el-button type="primary" @click="handleCreateBackup">
            {{ $t('backup.create') }}
          </el-button>
        </div>
      </template>
      <div class="p-4">
        <el-table v-loading="loading" :data="backups" style="width: 100%">
          <el-table-column prop="name" label="{{ $t('backup.fileName') }}" width="300">
            <template #default="{ row }">
              {{ row.name }}
            </template>
          </el-table-column>
          <el-table-column prop="size" label="{{ $t('backup.size') }}" width="120">
            <template #default="{ row }">
              {{ formatSize(row.size) }}
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="{{ $t('backup.createdAt') }}" width="200">
            <template #default="{ row }">
              {{ row.created_at }}
            </template>
          </el-table-column>
          <el-table-column label="{{ $t('backup.action') }}" width="200" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" @click="handleRestoreBackup(row.path)" :loading="restoring === row.path">
                {{ $t('backup.restore') }}
              </el-button>
              <el-button size="small" type="danger" @click="handleDeleteBackup(row.path)" :loading="deleting === row.path" style="margin-left: 8px">
                {{ $t('backup.delete') }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <div v-if="backups.length === 0" class="text-center py-8 text-gray-500">
          {{ $t('backup.noBackups') }}
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { createBackup, listBackups, deleteBackup, restoreBackup } from '@/api/backup'

const loading = ref(false)
const restoring = ref(null)
const deleting = ref(null)
const backups = ref([])

const loadBackups = async () => {
  loading.value = true
  try {
    const res = await listBackups()
    if (res.code === 200) {
      backups.value = res.data.backups
    }
  } catch (error) {
    ElMessage.error($t('backup.loadError'))
  } finally {
    loading.value = false
  }
}

const handleCreateBackup = async () => {
  try {
    const res = await createBackup()
    if (res.code === 200) {
      ElMessage.success($t('backup.createSuccess'))
      await loadBackups()
    }
  } catch (error) {
    ElMessage.error($t('backup.createError'))
  }
}

const handleDeleteBackup = async (backupPath) => {
  try {
    await ElMessageBox.confirm(
      $t('backup.deleteConfirm'),
      $t('backup.deleteConfirmTitle'),
      {
        confirmButtonText: $t('backup.confirm'),
        cancelButtonText: $t('backup.cancel'),
        type: 'warning'
      }
    )
    
    deleting.value = backupPath
    const res = await deleteBackup(backupPath)
    if (res.code === 200) {
      ElMessage.success($t('backup.deleteSuccess'))
      await loadBackups()
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error($t('backup.deleteError'))
    }
  } finally {
    deleting.value = null
  }
}

const handleRestoreBackup = async (backupPath) => {
  try {
    await ElMessageBox.confirm(
      $t('backup.restoreConfirm'),
      $t('backup.restoreConfirmTitle'),
      {
        confirmButtonText: $t('backup.confirm'),
        cancelButtonText: $t('backup.cancel'),
        type: 'warning'
      }
    )
    
    restoring.value = backupPath
    const res = await restoreBackup(backupPath)
    if (res.code === 200) {
      ElMessage.success($t('backup.restoreSuccess'))
      // 恢复后建议刷新页面或重启服务
      await ElMessageBox.alert(
        $t('backup.restoreNotice'),
        $t('backup.restoreNoticeTitle'),
        {
          confirmButtonText: $t('backup.confirm')
        }
      )
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error($t('backup.restoreError'))
    }
  } finally {
    restoring.value = null
  }
}

const formatSize = (size) => {
  if (size < 1024) {
    return size + ' B'
  } else if (size < 1024 * 1024) {
    return (size / 1024).toFixed(2) + ' KB'
  } else {
    return (size / (1024 * 1024)).toFixed(2) + ' MB'
  }
}

onMounted(() => {
  loadBackups()
})
</script>

<style scoped>
.backup-container {
  padding: 20px;
}
</style>
