<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { Bot } from 'lucide-vue-next'
import { useAuthStore } from '../stores/auth'
import { getBootstrapStatus } from '../api/auth'
import { translateApiError } from '../utils/error'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const { t } = useI18n()

const mode = ref<'login' | 'register'>('login')
const loading = ref(false)
const formRef = ref<FormInstance>()
const form = reactive({
  username: '',
  password: '',
  displayName: '',
})

const rules = computed<FormRules>(() => ({
  username: [
    { required: true, message: t('validation.usernameRequired'), trigger: 'blur' },
    { min: 3, max: 32, message: t('validation.usernameLength'), trigger: 'blur' },
  ],
  password: [
    { required: true, message: t('validation.passwordRequired'), trigger: 'blur' },
    { min: 8, message: t('validation.passwordLength'), trigger: 'blur' },
  ],
}))

onMounted(async () => {
  try {
    const status = await getBootstrapStatus()
    if (status.registration_open) {
      mode.value = 'register'
    }
  } catch {
    // Keep login mode when bootstrap status is unavailable.
  }
})

function switchMode() {
  mode.value = mode.value === 'login' ? 'register' : 'login'
  formRef.value?.clearValidate()
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    if (mode.value === 'login') {
      await auth.login(form.username.trim(), form.password)
      ElMessage.success(t('login.loginSuccess'))
    } else {
      await auth.register(form.username.trim(), form.password, form.displayName.trim())
      ElMessage.success(t('login.registerSuccess'))
    }
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/projects'
    await router.replace(redirect)
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <el-card class="login-card" shadow="never">
      <div class="login-brand">
        <div class="logo"><Bot :size="20" /></div>
        <div class="brand-copy">
          <h1>{{ t('common.appName') }}</h1>
          <p>{{ t('login.subtitle') }}</p>
        </div>
      </div>

      <el-form
        ref="formRef"
        class="login-form"
        :model="form"
        :rules="rules"
        label-position="top"
        size="large"
        @submit.prevent="submit"
      >
        <el-form-item :label="t('login.username')" prop="username">
          <el-input
            v-model="form.username"
            :placeholder="t('login.usernamePlaceholder')"
            autocomplete="username"
            autofocus
          />
          <el-text size="small" type="info">{{ t('login.usernameHelp') }}</el-text>
        </el-form-item>

        <el-form-item v-if="mode === 'register'" :label="t('login.displayName')" prop="displayName">
          <el-input v-model="form.displayName" :placeholder="t('login.displayNamePlaceholder')" />
          <el-text size="small" type="info">{{ t('login.displayNameHelp') }}</el-text>
        </el-form-item>

        <el-form-item :label="t('login.password')" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            show-password
            :placeholder="t('login.passwordPlaceholder')"
            :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
          />
          <el-text size="small" type="info">{{ t('login.passwordHelp') }}</el-text>
        </el-form-item>

        <el-button class="submit" type="primary" native-type="submit" :loading="loading">
          {{ mode === 'login' ? t('login.loginButton') : t('login.registerButton') }}
        </el-button>
      </el-form>

      <el-alert
        v-if="mode === 'register'"
        class="first-run-tip"
        :title="t('login.firstRunTip')"
        type="info"
        :closable="false"
        show-icon
      />

      <div class="switch-mode">
        <el-button link type="primary" @click="switchMode">
          {{ mode === 'login' ? t('login.needAccount') : t('login.haveAccount') }}
        </el-button>
      </div>
    </el-card>
  </div>
</template>

<style scoped lang="scss">
.login-page {
  display: grid;
  height: 100%;
  padding: 32px;
  overflow-y: auto;
  place-items: center;
  background: var(--act-bg);

  .login-card {
    width: min(440px, 100%);
    border-color: var(--act-border);
    border-radius: var(--act-radius-lg);
    background: var(--act-surface);

    :deep(.el-card__body) {
      padding: 32px;
    }

    .login-brand {
      display: flex;
      gap: 14px;
      align-items: center;
      margin-bottom: 28px;

      .logo {
        display: grid;
        width: 42px;
        height: 42px;
        flex: 0 0 auto;
        place-items: center;
        border-radius: var(--act-radius-md);
        color: #fff;
        background: var(--act-primary);
      }

      .brand-copy {
        h1 {
          margin: 0;
          font-size: 19px;
          font-weight: 600;
        }

        p {
          margin: 4px 0 0;
          color: var(--act-text-muted);
          font-size: 12px;
        }
      }
    }

    .login-form {
      :deep(.el-form-item) {
        margin-bottom: 22px;
      }

      :deep(.el-text) {
        margin-top: 8px;
        line-height: 1.55;
      }

      .submit {
        width: 100%;
        margin-top: 8px;
      }
    }

    .first-run-tip {
      margin-top: 18px;
    }

    .switch-mode {
      display: flex;
      justify-content: center;
      margin-top: 12px;
    }
  }
}
</style>

