<template>
  <el-dialog :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)"
    title="Excel 导入" width="450px">
    <div v-if="!result" class="upload-area">
      <el-upload ref="uploadRef" drag :auto-upload="false" :limit="1" accept=".xlsx,.xls"
        :on-change="onFileChange" :on-remove="onFileRemove">
        <el-icon style="font-size: 48px; color: #c0c4cc;"><UploadFilled /></el-icon>
        <div style="margin-top: 8px;">将 .xlsx 文件拖到此处，或点击上传</div>
      </el-upload>
      <el-button type="primary" style="margin-top: 16px; width: 100%;" :loading="uploading"
        :disabled="!selectedFile" @click="doImport">
        开始导入
      </el-button>
    </div>
    <div v-else class="result-area">
      <el-result icon="success" title="导入完成" :sub-title="`新增 ${result.imported_products} 个商品，导入 ${result.imported_prices} 条价格记录`">
        <template #extra>
          <el-button type="primary" @click="close">确定</el-button>
        </template>
      </el-result>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const props = defineProps({ modelValue: Boolean })
const emit = defineEmits(['update:modelValue', 'saved'])

const selectedFile = ref(null)
const uploading = ref(false)
const result = ref(null)

const onFileChange = (file) => {
  selectedFile.value = file.raw
}

const onFileRemove = () => {
  selectedFile.value = null
}

const doImport = async () => {
  if (!selectedFile.value) return
  uploading.value = true
  try {
    const fd = new FormData()
    fd.append('file', selectedFile.value)
    const data = await api.importExcel(fd)
    result.value = data
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '导入失败')
  } finally {
    uploading.value = false
  }
}

const close = () => {
  result.value = null
  selectedFile.value = null
  emit('update:modelValue', false)
  emit('saved')
}
</script>

<style scoped>
.upload-area {
  text-align: center;
}
.result-area {
  padding: 10px 0;
}
</style>
