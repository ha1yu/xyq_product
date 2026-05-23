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
                <el-dropdown-item @click="handleLogout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </el-header>
    <el-main class="app-main">
      <div class="dashboard-toolbar">
        <el-input v-model="keyword" placeholder="搜索商品名称..." prefix-icon="Search" clearable
          style="width: 300px" @input="handleSearch" />
      </div>
      <el-tabs v-model="activeCategory" @tab-change="handleCategoryChange">
        <el-tab-pane label="全部" name="all" />
        <el-tab-pane v-for="cat in categories" :key="cat.id" :label="cat.name" :name="String(cat.id)" />
      </el-tabs>
      <div class="dashboard-content" v-loading="loading" element-loading-text="加载中..." element-loading-background="rgba(245,245,245,0.8)">
        <el-empty v-if="showEmptyState" description="暂无商品数据" />
        <el-empty v-else-if="showNoResults" description="未找到匹配的商品" />
        <div v-else class="product-grid">
          <div v-for="product in filteredProducts" :key="product.id" class="product-card" @click="showTrend(product)">
            <div class="card-header">
              <span class="card-id">#{{ product.id }}</span>
              <el-tag size="small" effect="plain">{{ product.category_name }}</el-tag>
            </div>
            <div class="card-body">
              <div class="card-name">{{ product.name }}</div>
              <div v-if="product.remark" class="card-remark">{{ product.remark }}</div>
            </div>
            <div class="card-footer">
              <div v-if="product.prices && product.prices.length > 0" class="price-group">
                <div v-for="(p, i) in product.prices" :key="i" class="price-item">
                  <span class="price-val">{{ p.price }} 万两</span>
                  <span class="price-date">{{ p.date }}</span>
                </div>
              </div>
              <span v-else class="price-na">暂无价格记录</span>
            </div>
          </div>
        </div>
      </div>
      <PriceTrendDialog v-model="trendVisible" :product="trendProduct" />
    </el-main>
  </el-container>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useUserStore } from '../stores/user'
import api from '../api'
import PriceTrendDialog from '../components/PriceTrendDialog.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const currentRoute = computed(() => route.path)
const categories = ref([])
const products = ref([])
const keyword = ref('')
const activeCategory = ref('all')
const loading = ref(false)
const trendVisible = ref(false)
const trendProduct = ref(null)
let searchTimer = null

const filteredProducts = computed(() => {
  let list = products.value
  if (keyword.value) {
    list = list.filter(p => p.name.includes(keyword.value))
  }
  return list
})

const showEmptyState = computed(() => !loading.value && products.value.length === 0)
const showNoResults = computed(() => !loading.value && products.value.length > 0 && filteredProducts.value.length === 0)

const loadData = async () => {
  loading.value = true
  try {
    const [cats, prods] = await Promise.all([
      api.getCategories(),
      api.getProducts()
    ])
    categories.value = cats
    products.value = prods
  } catch (e) {
    ElMessage.error('加载数据失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(async () => {
    const params = {}
    if (keyword.value) params.keyword = keyword.value
    if (activeCategory.value !== 'all') params.category_id = activeCategory.value
    loading.value = true
    try {
      products.value = await api.getProducts(params)
    } finally {
      loading.value = false
    }
  }, 300)
}

const handleCategoryChange = async (tab) => {
  const params = {}
  if (tab !== 'all') params.category_id = tab
  if (keyword.value) params.keyword = keyword.value
  loading.value = true
  try {
    products.value = await api.getProducts(params)
  } finally {
    loading.value = false
  }
}

const handleLogout = () => {
  userStore.logout()
  ElMessage.success('已退出登录')
}

const showTrend = (product) => {
  trendProduct.value = product
  trendVisible.value = true
}

onMounted(loadData)
</script>

<style scoped>
.app-layout {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #f5f5f5;
}
.app-header {
  background: #1a1a2e;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  height: 60px;
}
.header-left .logo {
  color: #ffd700;
  font-size: 20px;
  font-weight: bold;
}
.header-right {
  display: flex;
  align-items: center;
  gap: 20px;
}
.header-right .el-menu {
  border-bottom: none;
}
.user-area {
  color: #e0e0e0;
}
.user-name {
  color: #e0e0e0;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
}
.app-main {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
  width: 100%;
  box-sizing: border-box;
  overflow: auto;
}
.dashboard-toolbar {
  margin-bottom: 15px;
  display: flex;
  justify-content: flex-end;
}
.price-val {
  color: #e6a23c;
  font-weight: bold;
}
.price-na {
  color: #c0c4cc;
}
.price-group {
  display: flex;
  gap: 8px;
}
.price-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 0 0 65px;
  padding: 2px 4px;
}
.price-item:last-child {
  border-right: none;
}
.price-date {
  font-size: 11px;
  color: #909399;
  margin-top: 2px;
}

/* Card layout */
.dashboard-content {
  position: relative;
  min-height: 200px;
}

.product-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.product-card {
  background: #fff;
  border-radius: 8px;
  padding: 16px 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: 1px solid #ebeef5;
  transition: box-shadow 0.25s ease, transform 0.25s ease;
  display: flex;
  flex-direction: column;
  cursor: pointer;
}

.product-card:hover {
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.1);
  transform: translateY(-3px);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid #f0f0f0;
}

.card-id {
  font-size: 12px;
  color: #909399;
  font-family: 'Courier New', monospace;
}

.card-body {
  flex: 1;
  margin-bottom: 12px;
}

.card-name {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  line-height: 1.45;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.card-remark {
  font-size: 13px;
  color: #909399;
  margin-top: 8px;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.card-footer {
  padding-top: 10px;
  border-top: 1px solid #f0f0f0;
}
</style>
