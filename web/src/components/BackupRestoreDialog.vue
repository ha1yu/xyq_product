<template>
  <el-dialog :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)"
    title="数据恢复" width="450px">
    <div v-if="!result" class="restore-area">
      <el-alert type="error" :closable="false" show-icon style="margin-bottom: 16px;">
        <template #title>
          <strong>警告：此操作将清除当前所有数据并替换为备份文件中的内容！</strong>
        </template>
      </el-alert>
      <el-upload ref="uploadRef" drag :auto-upload="false" :limit="1" accept=".xlsx"
        :on-change="onFileChange" :on-remove="onFileRemove">
        <el-icon style="font-size: 48px; color: #c0c4cc;"><UploadFilled /></el-icon>
        <div style="margin-top: 8px;">将备份 .xlsx 文件拖到此处，或点击上传</div>
      </el-upload>
      <el-button type="danger" style="margin-top: 16px; width: 100%;" :loading="restoring"
        :disabled="!selectedFile" @click="confirmRestore">
        确认恢复数据
      </el-button>
    </div>
    <div v-else class="result-area">
      <el-result icon="success" title="恢复完成"
        :sub-title="`已恢复 ${result.restored_categories} 个分类、${result.restored_products} 个商品、${result.restored_prices} 条价格记录`">
        <template #extra>
          <el-button type="primary" @click="close">确定</el-button>
        </template>
      </el-result>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'

const props = defineProps({ modelValue: Boolean })
const emit = defineEmits(['update:modelValue', 'restored'])

const selectedFile = ref(null)
const restoring = ref(false)
const result = ref(null)

const onFileChange = (file) => {
  selectedFile.value = file.raw
}

const onFileRemove = () => {
  selectedFile.value = null
}

const confirmRestore = async () => {
  if (!selectedFile.value) return
  try {
    await ElMessageBox.confirm(
      '此操作将永久覆盖当前所有数据，是否继续？',
      '危险操作',
      { type: 'error', confirmButtonText: '确认覆盖', cancelButtonText: '取消' }
    )
  } catch {
    return
  }

  restoring.value = true
  try {
    const fd = new FormData()
    fd.append('file', selectedFile.value)
    const data = await api.restoreBackup(fd)
    result.value = data
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '恢复失败')
  } finally {
    restoring.value = false
  }
}

const close = () => {
  result.value = null
  selectedFile.value = null
  emit('update:modelValue', false)
  emit('restored')
}
</script>

<style scoped>
.restore-area {
  text-align: center;
}
.result-area {
  padding: 10px 0;
}
</style>
