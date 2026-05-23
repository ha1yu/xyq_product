<template>
  <el-dialog :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)"
    :title="product?.name + ' - 价格走势'" width="750px" top="5vh" :close-on-click-modal="false">
    <div class="trend-dialog-body" v-loading="loading" element-loading-text="加载中...">
      <div v-if="!loading && priceCount > 0" class="stats-bar">
        <div class="stat-item">
          <span class="stat-label">价格记录</span>
          <span class="stat-value">{{ priceCount }} 条</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">平均价格</span>
          <span class="stat-value highlight">{{ averagePrice }} 万两</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">最低</span>
          <span class="stat-value">{{ minPrice }} 万两</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">最高</span>
          <span class="stat-value">{{ maxPrice }} 万两</span>
        </div>
      </div>
      <v-chart v-if="chartOption" :option="chartOption" autoresize style="height: 360px; width: 100%;" />
      <el-empty v-if="!loading && priceCount === 0" description="暂无价格记录" />
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import api from '../api'

use([CanvasRenderer, LineChart, TitleComponent, TooltipComponent, GridComponent])

const props = defineProps({
  modelValue: Boolean,
  product: Object
})

const emit = defineEmits(['update:modelValue'])

const loading = ref(false)
const chartOption = ref(null)
const priceCount = ref(0)
const averagePrice = ref(0)
const minPrice = ref(0)
const maxPrice = ref(0)

const colors = ['#5470c6']

const loadTrendData = async () => {
  if (!props.product) return
  loading.value = true
  chartOption.value = null
  try {
    const data = await api.getTrend(String(props.product.id))
    const series = data && data.length > 0 ? data[0] : null
    if (series && series.data && series.data.length > 0) {
      const dates = series.data.map(d => d.date)
      const prices = series.data.map(d => d.price)
      const nums = series.data.map(d => d.price)
      const sum = nums.reduce((a, b) => a + b, 0)

      priceCount.value = nums.length
      averagePrice.value = Math.round(sum / nums.length)
      minPrice.value = Math.min(...nums)
      maxPrice.value = Math.max(...nums)

      chartOption.value = {
        tooltip: {
          trigger: 'axis',
          valueFormatter: (val) => val + ' 万两'
        },
        grid: { top: 25, left: 60, right: 50, bottom: 40 },
        xAxis: { type: 'category', data: dates, boundaryGap: false },
        yAxis: { type: 'value', name: '万两', nameGap: 10 },
        series: [{
          type: 'line',
          data: prices,
          smooth: true,
          connectNulls: true,
          areaStyle: { opacity: 0.15 },
          itemStyle: { color: colors[0] },
          lineStyle: { color: colors[0] }
        }]
      }
    } else {
      priceCount.value = 0
    }
  } catch (e) {
    priceCount.value = 0
  } finally {
    loading.value = false
  }
}

watch(() => props.modelValue, (val) => {
  if (val) {
    loadTrendData()
  }
})
</script>

<style scoped>
.trend-dialog-body {
  min-height: 200px;
}
.stats-bar {
  display: flex;
  gap: 20px;
  margin-bottom: 16px;
  padding: 12px 16px;
  background: #f9f9f9;
  border-radius: 8px;
}
.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
}
.stat-label {
  font-size: 12px;
  color: #909399;
  margin-bottom: 4px;
}
.stat-value {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  white-space: nowrap;
}
.stat-value.highlight {
  color: #e6a23c;
}
</style>
