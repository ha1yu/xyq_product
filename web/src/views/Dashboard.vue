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
          <template v-if="userStore.isLoggedIn">
            <el-dropdown>
              <span class="user-name">
                <el-icon><User /></el-icon> 管理员
              </span>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="handleLogout">退出登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
          <template v-else>
            <el-button type="primary" size="small" @click="$router.push('/login')">登录</el-button>
          </template>
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
      <el-table :data="filteredProducts" stripe border style="width:100%"
        :default-sort="{ prop: 'id', order: 'ascending' }" max-height="calc(100vh - 200px)">
        <el-table-column prop="id" label="ID" width="60" sortable />
        <el-table-column prop="name" label="商品名称" min-width="150" />
        <el-table-column prop="category_name" label="分类" width="100" />
        <el-table-column label="最新价格" width="120">
          <template #default="{ row }">
            <span v-if="row.latestPrice !== null" class="price-val">
              {{ row.latestPrice }} 万两
            </span>
            <span v-else class="price-na">--</span>
          </template>
        </el-table-column>
        <el-table-column prop="latestDate" label="更新日期" width="120" />
        <el-table-column prop="remark" label="备注" min-width="150" show-overflow-tooltip />
      </el-table>
    </el-main>
  </el-container>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useUserStore } from '../stores/user'
import api from '../api'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const currentRoute = computed(() => route.path)
const categories = ref([])
const products = ref([])
const keyword = ref('')
const activeCategory = ref('all')
let searchTimer = null

const filteredProducts = computed(() => {
  let list = products.value
  if (keyword.value) {
    list = list.filter(p => p.name.includes(keyword.value))
  }
  return list
})

const loadData = async () => {
  try {
    const [cats, prods] = await Promise.all([
      api.getCategories(),
      api.getProducts()
    ])
    categories.value = cats
    products.value = prods
    // Load latest prices for each product
    await loadLatestPrices()
  } catch (e) {
    ElMessage.error('加载数据失败')
  }
}

const loadLatestPrices = async () => {
  // Batch load prices - we'll get them all at once via trend API
  // For now, show products without prices initially
}

const handleSearch = () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(async () => {
    const params = {}
    if (keyword.value) params.keyword = keyword.value
    if (activeCategory.value !== 'all') params.category_id = activeCategory.value
    products.value = await api.getProducts(params)
  }, 300)
}

const handleCategoryChange = async (tab) => {
  const params = {}
  if (tab !== 'all') params.category_id = tab
  if (keyword.value) params.keyword = keyword.value
  products.value = await api.getProducts(params)
}

const handleLogout = () => {
  userStore.logout()
  ElMessage.success('已退出登录')
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
  overflow: hidden;
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
</style>
