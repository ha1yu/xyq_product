<template>
  <el-container class="app-layout">
    <el-header class="app-header">
      <div class="header-left">
        <span class="logo">⚔️ 梦幻物价追踪</span>
      </div>
      <div class="header-right">
        <el-menu mode="horizontal" :default-active="currentRoute" router :ellipsis="false"
          background-color="#1a1a2e" text-color="#e0e0e0" active-text-color="#ffd700">
          <el-menu-item index="/">价格总览</el-menu-item>
          <el-menu-item index="/products">商品管理</el-menu-item>
          <el-menu-item index="/trend">走势分析</el-menu-item>
        </el-menu>
        <div class="user-area">
          <el-dropdown>
            <span class="user-name"><el-icon><User /></el-icon> 管理员</span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="settingsVisible = true">系统设置</el-dropdown-item>
                <el-dropdown-item @click="handleLogout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </el-header>
    <el-main class="app-main">
      <div class="page-title">
        <h2>商品管理</h2>
        <div class="title-actions">
          <el-button type="success" @click="ocrDialogVisible = true" >
            <el-icon><Camera /></el-icon> 拍照识别
          </el-button>
          <el-button type="primary" @click="showAddDialog" >
            <el-icon><Plus /></el-icon> 新增商品
          </el-button>
          <el-button type="warning" @click="importDialogVisible = true">
            <el-icon><Upload /></el-icon> Excel导入
          </el-button>
        </div>
      </div>
      <div class="filter-bar">
        <el-select v-model="filterCategory" placeholder="筛选分类" clearable style="width: 160px"
          @change="loadProducts">
          <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
        </el-select>
        <el-input v-model="filterKeyword" placeholder="搜索商品" prefix-icon="Search" clearable
          style="width: 250px" @input="debouncedLoad" />
      </div>
      <div class="table-wrapper">
        <el-table :data="products" stripe border height="100%" style="width:100%">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="name" label="商品名称" min-width="150" />
          <el-table-column prop="category_name" label="分类" width="100" />
          <el-table-column prop="remark" label="备注" min-width="150" show-overflow-tooltip />
          <el-table-column label="操作" width="260" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="showPrices(row)">价格记录</el-button>
              <el-button size="small" type="primary" @click="editProduct(row)"
                >编辑</el-button>
              <el-button size="small" type="danger" @click="deleteProduct(row)"
                >删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-main>

    <!-- Add/Edit Dialog -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑商品' : '新增商品'" width="450px">
      <el-form :model="productForm" :rules="productRules" ref="productFormRef" label-width="80px">
        <el-form-item label="商品名称" prop="name">
          <el-input v-model="productForm.name" />
        </el-form-item>
        <el-form-item label="分类" prop="category_id">
          <el-select v-model="productForm.category_id" style="width:100%">
            <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="productForm.remark" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitProduct" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <!-- Prices Dialog -->
    <el-dialog v-model="pricesVisible" :title="priceProduct?.name + ' - 价格记录'" width="600px">
      <div style="margin-bottom:10px;">
        <el-button type="primary" size="small" @click="showAddPrice"
          >添加价格</el-button>
      </div>
      <el-table :data="prices" stripe border size="small">
        <el-table-column prop="price_date" label="日期" width="130" />
        <el-table-column prop="price" label="价格（万两）" width="130" />
        <el-table-column label="操作" width="80">
          <template #default="{ row }">
            <el-button size="small" type="danger" link @click="deletePrice(row)"
              >删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="addPriceVisible" style="margin-top:15px; padding:15px; background:#f5f5f5; border-radius:6px;">
        <el-form :inline="true">
          <el-form-item label="日期">
            <el-date-picker v-model="newPrice.date" type="date" value-format="YYYY-MM-DD"
              placeholder="选择日期" style="width:160px" />
          </el-form-item>
          <el-form-item label="价格">
            <el-input-number v-model="newPrice.price" :precision="1" :step="0.5" :min="0"
              style="width:130px" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="submitPrice">确定</el-button>
            <el-button @click="addPriceVisible = false">取消</el-button>
          </el-form-item>
        </el-form>
      </div>
    </el-dialog>

    <!-- OCR Dialog -->
    <OcrDialog v-model="ocrDialogVisible" :products="products" @saved="onOcrSaved" />
    <ExcelImportDialog v-model="importDialogVisible" @saved="loadProducts" />
    <SettingsDialog v-model="settingsVisible" />
  </el-container>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '../stores/user'
import api from '../api'
import OcrDialog from '../components/OcrDialog.vue'
import ExcelImportDialog from '../components/ExcelImportDialog.vue'
import SettingsDialog from '../components/SettingsDialog.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const currentRoute = computed(() => route.path)
const categories = ref([])
const products = ref([])
const filterCategory = ref(null)
const filterKeyword = ref('')

const dialogVisible = ref(false)
const isEdit = ref(false)
const editingId = ref(null)
const submitting = ref(false)
const productFormRef = ref(null)
const productForm = ref({ name: '', category_id: null, remark: '' })
const productRules = {
  name: [{ required: true, message: '请输入商品名称', trigger: 'blur' }],
  category_id: [{ required: true, message: '请选择分类', trigger: 'change' }]
}

const pricesVisible = ref(false)
const prices = ref([])
const priceProduct = ref(null)
const addPriceVisible = ref(false)
const newPrice = ref({ date: '', price: 0 })

const ocrDialogVisible = ref(false)
const importDialogVisible = ref(false)
const settingsVisible = ref(false)

let loadTimer = null
const debouncedLoad = () => {
  clearTimeout(loadTimer)
  loadTimer = setTimeout(loadProducts, 300)
}

const loadProducts = async () => {
  const params = {}
  if (filterCategory.value) params.category_id = filterCategory.value
  if (filterKeyword.value) params.keyword = filterKeyword.value
  products.value = await api.getProducts(params)
}

const loadCategories = async () => {
  categories.value = await api.getCategories()
}

const showAddDialog = () => {
  isEdit.value = false
  editingId.value = null
  productForm.value = { name: '', category_id: null, remark: '' }
  dialogVisible.value = true
}

const editProduct = (row) => {
  isEdit.value = true
  editingId.value = row.id
  productForm.value = { name: row.name, category_id: row.category_id, remark: row.remark || '' }
  dialogVisible.value = true
}

const submitProduct = async () => {
  const valid = await productFormRef.value.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    if (isEdit.value) {
      await api.updateProduct(editingId.value, productForm.value)
      ElMessage.success('修改成功')
    } else {
      await api.createProduct(productForm.value)
      ElMessage.success('新增成功')
    }
    dialogVisible.value = false
    loadProducts()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '操作失败')
  } finally {
    submitting.value = false
  }
}

const deleteProduct = async (row) => {
  await ElMessageBox.confirm(`确定删除商品"${row.name}"？`, '提示', { type: 'warning' })
  await api.deleteProduct(row.id)
  ElMessage.success('删除成功')
  loadProducts()
}

const showPrices = async (row) => {
  priceProduct.value = row
  prices.value = await api.getPrices(row.id)
  pricesVisible.value = true
  addPriceVisible.value = false
}

const showAddPrice = () => {
  newPrice.value = { date: '', price: 0 }
  addPriceVisible.value = true
}

const submitPrice = async () => {
  if (!newPrice.value.date || !newPrice.value.price) {
    ElMessage.warning('请填写完整')
    return
  }
  await api.createPrice({ product_id: priceProduct.value.id, ...newPrice.value })
  ElMessage.success('添加成功')
  prices.value = await api.getPrices(priceProduct.value.id)
  addPriceVisible.value = false
}

const deletePrice = async (row) => {
  await ElMessageBox.confirm('确定删除该价格记录？', '提示', { type: 'warning' })
  await api.deletePrice(row.id)
  ElMessage.success('删除成功')
  prices.value = await api.getPrices(priceProduct.value.id)
}

const onOcrSaved = () => {
  loadProducts()
}

const handleLogout = () => {
  userStore.logout()
  ElMessage.success('已退出登录')
  router.push('/')
}

onMounted(() => {
  loadCategories()
  loadProducts()
})
</script>

<style scoped>
.app-layout { height: 100vh; display: flex; flex-direction: column; background: #f5f5f5; overflow: hidden; }
.app-header {
  background: #1a1a2e; display: flex; align-items: center;
  justify-content: space-between; padding: 0 20px; height: 60px; flex-shrink: 0;
}
.header-left .logo { color: #ffd700; font-size: 20px; font-weight: bold; }
.header-right { display: flex; align-items: center; gap: 20px; }
.header-right .el-menu { border-bottom: none; }
.user-area { color: #e0e0e0; }
.user-name { color: #e0e0e0; cursor: pointer; display: flex; align-items: center; gap: 4px; }
.app-main {
  padding: 20px; max-width: 1400px; margin: 0 auto; width: 100%; box-sizing: border-box;
  flex: 1; overflow: hidden; display: flex; flex-direction: column;
}
.page-title { display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px; flex-shrink: 0; }
.page-title h2 { margin: 0; color: #303133; }
.title-actions { display: flex; gap: 10px; }
.filter-bar { display: flex; gap: 12px; margin-bottom: 15px; flex-shrink: 0; }
.table-wrapper { flex: 1; overflow: hidden; }
</style>
