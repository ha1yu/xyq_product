<template>
  <el-table :data="prices" stripe border size="small">
    <el-table-column prop="price_date" label="日期" width="130" />
    <el-table-column prop="price" label="价格（万两）" width="130" />
    <el-table-column label="操作" width="80">
      <template #default="{ row }">
        <el-button size="small" type="danger" link @click="$emit('delete', row)">删除</el-button>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup>
defineProps({ prices: Array })
defineEmits(['delete'])
</script>
