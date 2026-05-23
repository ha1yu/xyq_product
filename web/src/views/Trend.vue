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
      <h2 style="margin-top:0;">📈 走势分析</h2>
      <div class="trend-controls">
        <el-select v-model="selectedProducts" multiple filterable placeholder="选择商品（支持多选对比）"
          style="width: 400px" @change="loadTrend">
          <el-option-group v-for="cat in categoriesWithProducts" :key="cat.id" :label="cat.name">
            <el-option v-for="p in cat.products" :key="p.id" :label="p.name" :value="p.id" />
          </el-option-group>
        </el-select>
        <el-button type="primary" @click="loadTrend" :disabled="selectedProducts.length === 0">
          查看走势
        </el-button>
      </div>
      <div class="chart-container" v-if="chartOption">
        <v-chart :option="chartOption" autoresize style="height: 500px; width: 100%;" />
      </div>
      <el-empty v-else description="请选择商品查看价格走势" />
    </el-main>
    <SettingsDialog v-model="settingsVisible" />
  </el-container>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { useUserStore } from '../stores/user'
import SettingsDialog from '../components/SettingsDialog.vue'
import api from '../api'

use([CanvasRenderer, LineChart, TitleComponent, TooltipComponent, LegendComponent, GridComponent])

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const currentRoute = computed(() => route.path)
const categories = ref([])
const products = ref([])
const selectedProducts = ref([])
const chartOption = ref(null)
const settingsVisible = ref(false)

const colors = ['#5470c6', '#91cc75', '#fac858', '#ee6666', '#73c0de', '#3ba272', '#fc8452', '#9a60b4']

const categoriesWithProducts = computed(() => {
  const map = {}
  categories.value.forEach(c => { map[c.id] = { ...c, products: [] } })
  products.value.forEach(p => {
    if (map[p.category_id]) map[p.category_id].products.push(p)
  })
  return Object.values(map)
})

const loadTrend = async () => {
  if (selectedProducts.value.length === 0) return
  try {
    const ids = selectedProducts.value.join(',')
    const data = await api.getTrend(ids)
    if (!data || data.length === 0) {
      ElMessage.warning('暂无价格数据')
      return
    }
    // Build all dates
    const dateSet = new Set()
    data.forEach(s => s.data.forEach(d => dateSet.add(d.date)))
    const dates = [...dateSet].sort()

    const series = data.map((s, i) => ({
      name: s.product_name,
      type: 'line',
      data: dates.map(d => {
        const point = s.data.find(p => p.date === d)
        return point ? point.price : null
      }),
      smooth: true,
      connectNulls: true,
      itemStyle: { color: colors[i % colors.length] }
    }))

    chartOption.value = {
      title: { text: '商品价格走势对比', left: 'center' },
      tooltip: {
        trigger: 'axis',
        formatter: (params) => {
          let html = `<b>${params[0].axisValue}</b><br/>`
          params.forEach(p => {
            html += `${p.marker} ${p.seriesName}: ${p.value != null ? p.value + ' 万两' : '--'}<br/>`
          })
          return html
        }
      },
      legend: { top: 30, data: data.map(s => s.product_name) },
      grid: { top: 70, left: 60, right: 30, bottom: 30 },
      xAxis: { type: 'category', data: dates, boundaryGap: false },
      yAxis: { type: 'value', name: '万两' },
      series
    }
  } catch (e) {
    ElMessage.error('加载走势数据失败')
  }
}

const loadData = async () => {
  const [cats, prods] = await Promise.all([api.getCategories(), api.getProducts()])
  categories.value = cats
  products.value = prods
}

const handleLogout = () => {
  userStore.logout()
  ElMessage.success('已退出登录')
  router.push('/')
}

onMounted(loadData)
</script>

<style scoped>
.app-layout { min-height: 100vh; background: #f5f5f5; }
.app-header {
  background: #1a1a2e; display: flex; align-items: center;
  justify-content: space-between; padding: 0 20px; height: 60px;
}
.header-left .logo { color: #ffd700; font-size: 20px; font-weight: bold; }
.header-right { display: flex; align-items: center; gap: 20px; }
.header-right .el-menu { border-bottom: none; }
.user-area { color: #e0e0e0; }
.user-name { color: #e0e0e0; cursor: pointer; display: flex; align-items: center; gap: 4px; }
.app-main { padding: 20px; max-width: 1400px; margin: 0 auto; width: 100%; box-sizing: border-box; }
.trend-controls { display: flex; gap: 12px; margin-bottom: 20px; }
.chart-container { background: #fff; border-radius: 8px; padding: 20px; box-shadow: 0 2px 8px rgba(0,0,0,0.06); }
</style>
