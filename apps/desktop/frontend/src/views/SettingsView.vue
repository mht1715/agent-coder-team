<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import { useThemeStore, type ThemeMode } from '../stores/theme'
import { isWailsRuntime } from '../api/client'
import { setLocale, type SupportedLocale } from '../i18n'

const { t, locale } = useI18n()
const auth = useAuthStore()
const theme = useThemeStore()
const router = useRouter()
const usingWailsIPC = isWailsRuntime()

const themeMode = computed({
  get: () => theme.mode,
  set: (value: ThemeMode) => theme.setMode(value),
})

const currentLocale = computed({
  get: () => locale.value as SupportedLocale,
  set: (value: SupportedLocale) => setLocale(value),
})

const themeOptions = computed(() => [
  { label: t('settings.dark'), value: 'dark' },
  { label: t('settings.light'), value: 'light' },
  { label: t('settings.system'), value: 'system' },
])

const languageOptions = computed(() => [
  { label: t('projects.languageZhCN'), value: 'zh-CN' },
  { label: t('projects.languageEnUS'), value: 'en-US' },
])

async function logout() {
  await auth.logout()
  ElMessage.success(t('common.logout'))
  await router.push('/login')
}
</script>

<template>
  <div class="settings-page">
    <div class="settings-head">
      <h2>{{ t('settings.title') }}</h2>
      <p>{{ t('settings.subtitle') }}</p>
    </div>

    <div class="settings-body">
      <el-card class="settings-section" shadow="never">
        <template #header>
          <strong>{{ t('settings.appearance') }}</strong>
        </template>

        <el-form label-position="top">
          <el-form-item :label="t('settings.themeMode')">
            <el-segmented v-model="themeMode" :options="themeOptions" />
            <el-text size="small" type="info">{{ t('settings.themeHelp') }}</el-text>
          </el-form-item>
          <el-form-item :label="t('settings.language')">
            <el-select v-model="currentLocale" class="language-select">
              <el-option
                v-for="option in languageOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </el-select>
            <el-text size="small" type="info">{{ t('settings.languageHelp') }}</el-text>
          </el-form-item>
        </el-form>
      </el-card>

      <el-card class="settings-section" shadow="never">
        <template #header>
          <strong>{{ t('settings.account') }}</strong>
        </template>

        <el-descriptions :column="1" border>
          <el-descriptions-item :label="t('settings.signedInAs')">
            {{ auth.user?.display_name || auth.user?.username }} (@{{ auth.user?.username }})
          </el-descriptions-item>
          <el-descriptions-item :label="t('settings.apiMode')">
            <div class="transport-row">
              <span>{{ usingWailsIPC ? t('settings.apiModeWails') : t('settings.apiModeHttp') }}</span>
              <el-tag :type="usingWailsIPC ? 'success' : 'warning'" effect="plain">
                {{ usingWailsIPC ? t('settings.desktopIpc') : t('settings.developmentMode') }}
              </el-tag>
            </div>
          </el-descriptions-item>
        </el-descriptions>

        <div class="account-actions">
          <el-text size="small" type="info">{{ t('settings.accountHelp') }}</el-text>
          <el-button type="danger" plain @click="logout">{{ t('common.logout') }}</el-button>
        </div>
      </el-card>
    </div>
  </div>
</template>

<style scoped lang="scss">
.settings-page {
  display: flex;
  max-width: 900px;
  height: 100%;
  margin: 0 auto;
  padding: 24px 28px 0;
  flex-direction: column;
  overflow: hidden;

  .settings-head {
    flex: 0 0 auto;
    margin-bottom: 20px;

    h2 {
      margin: 0;
      font-size: 21px;
      font-weight: 600;
    }

    p {
      margin: 7px 0 0;
      color: var(--act-text-muted);
      font-size: 12px;
    }
  }

  .settings-body {
    min-height: 0;
    flex: 1;
    padding: 2px 6px 28px 2px;
    overflow-y: auto;
  }

  .settings-section {
    margin-bottom: 18px;
    border-radius: var(--act-radius-lg);
    background: var(--act-surface);

    :deep(.el-card__header) {
      padding: 16px 20px;
    }

    :deep(.el-card__body) {
      padding: 20px;
    }

    :deep(.el-form-item) {
      margin-bottom: 22px;

      &:last-child {
        margin-bottom: 0;
      }
    }

    :deep(.el-form-item .el-text) {
      margin-top: 8px;
      line-height: 1.55;
    }

    .language-select {
      width: 220px;
    }

    .transport-row {
      display: flex;
      gap: 14px;
      align-items: center;
      justify-content: space-between;
    }

    .account-actions {
      display: flex;
      gap: 16px;
      align-items: center;
      justify-content: space-between;
      margin-top: 20px;
    }
  }
}
</style>
