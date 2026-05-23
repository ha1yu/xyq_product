<template>
  <el-dialog v-model="visible" title="📷 拍照识别录入" width="700px" :close-on-click-modal="false">
    <!-- Step 1: Upload image -->
    <div v-if="step === 1">
      <el-upload
        ref="uploadRef"
        :auto-upload="false"
        :show-file-list="false"
        accept="image/*"
        :on-change="handleFileChange"
        drag
      >
        <div class="upload-area">
          <el-icon :size="48" color="#909399"><UploadFilled /></el-icon>
          <p>将图片拖到此处，或 <em>点击上传</em></p>
          <p class="tip">支持 JPG / PNG / WEBP，建议截图或拍照梦幻西游摆摊/商品界面</p>
        </div>
      </el-upload>
      <div v-if="previewUrl" class="preview-box">
        <img :src="previewUrl" class="preview-img" />
        <div class="preview-actions">
          <el-button type="primary" @click="doOcr" :loading="loading">
            <el-icon><MagicStick /></el-icon> 识别图片
          </el-button>
          <el-button @click="clearImage">重新选择</el-button>
        </div>
      </div>
    </div>

    <!-- Step 2: Review and edit results -->
    <div v-if="step === 2">
      <div class="step-header">
        <el-button link @click="step = 1"><el-icon><ArrowLeft /></el-icon> 返回重新识别</el-button>
        <span>识别结果（{{ ocrItems.length }} 项），请确认后录入</span>
      </div>

      <div class="ocr-date-row">
        <span>价格日期：</span>
        <el-date-picker v-model="priceDate" type="date" value-format="YYYY-MM-DD"
          placeholder="选择日期" style="width: 180px" />
      </div>

      <el-table :data="ocrItems" border size="small" style="margin-top:10px;">
        <el-table-column label="商品名" min-width="140">
          <template #default="{ row }">
            <el-autocomplete v-model="row.name" :fetch-suggestions="queryProducts"
              placeholder="商品名" style="width:100%" @select="(item) => row.product_id = item.id" />
          </template>
        </el-table-column>
        <el-table-column label="价格（万两）" width="140">
          <template #default="{ row }">
            <el-input-number v-model="row.price" :precision="1" :step="0.5" :min="0" size="small"
              controls-position="right" style="width:100%" />
          </template>
        </el-table-column>
        <el-table-column label="匹配" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.product_id" type="success" size="small">已匹配</el-tag>
            <el-tag v-else type="warning" size="small">未匹配</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="60" align="center">
          <template #default="{ $index }">
            <el-button type="danger" link size="small" @click="ocrItems.splice($index, 1)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div style="margin-top:15px; text-align:right;">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="doBatchSave" :loading="saving" :disabled="matchedItems.length === 0">
          录入 {{ matchedItems.length }} 项
        </el-button>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const props = defineProps({
  modelValue: Boolean,
  products: Array
})
const emit = defineEmits(['update:modelValue', 'saved'])

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const step = ref(1)
const loading = ref(false)
const saving = ref(false)
const previewUrl = ref('')
const imageBase64 = ref('')
const ocrItems = ref([])
const priceDate = ref(new Date().toISOString().split('T')[0])

watch(visible, (v) => {
  if (v) {
    step.value = 1
    clearImage()
    ocrItems.value = []
  }
})

const handleFileChange = (file) => {
  const reader = new FileReader()
  reader.onload = (e) => {
    previewUrl.value = e.target.result
    imageBase64.value = e.target.result
  }
  reader.readAsDataURL(file.raw)
}

const clearImage = () => {
  previewUrl.value = ''
  imageBase64.value = ''
}

const doOcr = async () => {
  loading.value = true
  try {
    const res = await api.ocrRecognize({ image: imageBase64.value })
    const items = res.items || []
    if (items.length === 0) {
      ElMessage.warning('未识别到商品，请尝试更清晰的图片')
      return
    }
    // Try to match with existing products
    ocrItems.value = items.map(item => ({
      name: item.name,
      price: item.price,
      product_id: findProduct(item.name)
    }))
    step.value = 2
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '识别失败')
  } finally {
    loading.value = false
  }
}

const findProduct = (name) => {
  const n = name.trim()
  const exact = props.products.find(p => p.name === n)
  if (exact) return exact.id
  const partial = props.products.find(p => p.name.includes(n) || n.includes(p.name))
  return partial ? partial.id : null
}

const queryProducts = (queryString, cb) => {
  const results = props.products
    .filter(p => p.name.includes(queryString))
    .map(p => ({ value: p.name, id: p.id }))
  cb(results)
}

const matchedItems = computed(() =>
  ocrItems.value.filter(item => item.product_id && item.price > 0)
)

const doBatchSave = async () => {
  if (!priceDate.value) {
    ElMessage.warning('请选择价格日期')
    return
  }
  saving.value = true
  try {
    const items = matchedItems.value.map(item => ({
      product_id: item.product_id,
      price_date: priceDate.value,
      price: item.price
    }))
    await api.batchCreatePrices({ items })
    ElMessage.success(`成功录入 ${items.length} 条价格记录`)
    emit('saved')
    visible.value = false
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '录入失败')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.upload-area {
  padding: 40px 20px;
  text-align: center;
  color: #909399;
}
.upload-area p { margin: 8px 0; }
.upload-area em { color: #409eff; font-style: normal; }
.upload-area .tip { font-size: 12px; color: #c0c4cc; }
.preview-box { margin-top: 15px; text-align: center; }
.preview-img { max-width: 100%; max-height: 300px; border-radius: 6px; }
.preview-actions { margin-top: 10px; }
.step-header {
  display: flex; align-items: center; gap: 10px;
  margin-bottom: 10px; font-size: 14px; color: #606266;
}
.ocr-date-row {
  display: flex; align-items: center; gap: 8px;
  margin-top: 10px; font-size: 14px;
}
</style>
