<template>
  <el-dialog :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)"
    title="系统设置" width="500px">
    <el-tabs v-model="activeTab">
      <el-tab-pane label="OCR 配置" name="ocr">
        <el-form label-width="100px" style="margin-top: 16px;">
          <el-form-item label="服务地址">
            <el-input v-model="ocrForm.endpoint" placeholder="http://host:port/v1/chat/completions" />
          </el-form-item>
          <el-form-item label="模型名称">
            <el-input v-model="ocrForm.model" placeholder="模型名称" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="saveOcr" :loading="saving">保存</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>
      <el-tab-pane label="修改密码" name="password">
        <el-form label-width="100px" style="margin-top: 16px;">
          <el-form-item label="旧密码">
            <el-input v-model="pwdForm.oldPassword" type="password" show-password />
          </el-form-item>
          <el-form-item label="新密码">
            <el-input v-model="pwdForm.newPassword" type="password" show-password />
          </el-form-item>
          <el-form-item label="确认密码">
            <el-input v-model="pwdForm.confirmPassword" type="password" show-password />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="savePassword" :loading="saving">修改密码</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>
    </el-tabs>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useUserStore } from '../stores/user'
import api from '../api'

const props = defineProps({ modelValue: Boolean })
defineEmits(['update:modelValue'])

const router = useRouter()
const userStore = useUserStore()

const activeTab = ref('ocr')
const saving = ref(false)
const ocrForm = reactive({ endpoint: '', model: '' })
const pwdForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })

watch(() => props.modelValue, async (val) => {
  if (val) {
    activeTab.value = 'ocr'
    pwdForm.oldPassword = ''
    pwdForm.newPassword = ''
    pwdForm.confirmPassword = ''
    try {
      const data = await api.getSettings()
      ocrForm.endpoint = data.ocr_endpoint || ''
      ocrForm.model = data.ocr_model || ''
    } catch (e) { /* ignore */ }
  }
})

const saveOcr = async () => {
  saving.value = true
  try {
    await api.updateSettings({ ocr_endpoint: ocrForm.endpoint, ocr_model: ocrForm.model })
    ElMessage.success('OCR 配置已保存')
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

const savePassword = async () => {
  if (!pwdForm.oldPassword || !pwdForm.newPassword) {
    ElMessage.warning('请填写完整')
    return
  }
  if (pwdForm.newPassword !== pwdForm.confirmPassword) {
    ElMessage.warning('两次密码不一致')
    return
  }
  saving.value = true
  try {
    await api.changePassword({ old_password: pwdForm.oldPassword, new_password: pwdForm.newPassword })
    ElMessage.success('密码已修改，请重新登录')
    userStore.logout()
    router.push('/login')
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '修改失败')
  } finally {
    saving.value = false
  }
}
</script>
