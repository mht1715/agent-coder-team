<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Bot, FolderKanban, LogOut, Moon, Settings, Sun } from 'lucide-vue-next'
import { useAuthStore } from '../stores/auth'
import { useThemeStore } from '../stores/theme'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const theme = useThemeStore()
const { t } = useI18n()

const activeMenu = computed(() => {
  if (route.path.startsWith('/projects')) return '/projects'
  if (route.path.startsWith('/settings')) return '/settings'
  return '/projects'
})

const showBack = computed(() => route.name !== 'projects')

const pageTitle = computed(() => {
  if (route.name === 'settings') return t('settings.title')
  if (route.name === 'project') return t('chat.title')
  return t('common.projects')
})

const themeActionLabel = computed(() =>
  theme.resolvedTheme === 'dark' ? t('settings.light') : t('settings.dark'),
)

function toggleTheme() {
  theme.setMode(theme.resolvedTheme === 'dark' ? 'light' : 'dark')
}

function goBack() {
  void router.push({ name: 'projects' })
}

async function handleLogout() {
  await auth.logout()
  ElMessage.success(t('common.logout'))
  await router.push('/login')
}
</script>

<template>
  <el-container class="shell">
    <el-aside width="236px" class="sidebar">
      <div class="brand">
        <div class="brand-mark"><Bot :size="20" /></div>
        <div class="brand-copy">
          <strong>{{ t('common.appName') }}</strong>
          <span>{{ t('common.localWorkspace') }}</span>
        </div>
      </div>

      <el-menu :default-active="activeMenu" class="nav" router>
        <el-menu-item index="/projects">
          <el-icon><FolderKanban :size="18" /></el-icon>
          <span>{{ t('common.projects') }}</span>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><Settings :size="18" /></el-icon>
          <span>{{ t('common.settings') }}</span>
        </el-menu-item>
      </el-menu>

      <div class="sidebar-footer">
        <div class="account">
          <el-avatar :size="34" class="avatar">
            {{ auth.displayName.slice(0, 1).toUpperCase() }}
          </el-avatar>
          <div class="account-text">
            <strong>{{ auth.displayName }}</strong>
            <span>@{{ auth.user?.username }}</span>
          </div>
        </div>
        <el-tooltip :content="t('common.logout')" placement="top">
          <el-button circle text @click="handleLogout">
            <LogOut :size="17" />
          </el-button>
        </el-tooltip>
      </div>
    </el-aside>

    <el-container class="main">
      <el-header height="60px" class="topbar">
        <div class="topbar-left">
          <el-tooltip v-if="showBack" :content="t('common.backToProjects')" placement="bottom">
            <el-button circle text class="back-button" @click="goBack">
              <ArrowLeft :size="18" />
            </el-button>
          </el-tooltip>

          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ name: 'projects' }">{{ t('common.appName') }}</el-breadcrumb-item>
            <el-breadcrumb-item>{{ pageTitle }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>

        <div class="topbar-actions">
          <el-tooltip :content="themeActionLabel" placement="bottom">
            <el-button circle text @click="toggleTheme">
              <Sun v-if="theme.resolvedTheme === 'dark'" :size="18" />
              <Moon v-else :size="18" />
            </el-button>
          </el-tooltip>
        </div>
      </el-header>

      <el-main class="content">
        <router-view v-slot="{ Component, route: viewRoute }">
          <transition name="page-fade" mode="out-in">
            <keep-alive>
              <component :is="Component" :key="viewRoute.fullPath" />
            </keep-alive>
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<style scoped lang="scss">
.shell {
  height: 100%;
  background: var(--act-bg);

  .sidebar {
    display: flex;
    min-height: 0;
    flex-direction: column;
    padding: 18px 14px;
    border-right: 1px solid var(--act-border);
    background: var(--act-bg-soft);

    .brand {
      display: flex;
      gap: 12px;
      align-items: center;
      padding: 4px 8px 20px;

      .brand-mark {
        display: grid;
        width: 38px;
        height: 38px;
        flex: 0 0 auto;
        place-items: center;
        border-radius: var(--act-radius-md);
        color: #fff;
        background: var(--act-primary);
      }

      .brand-copy {
        display: flex;
        min-width: 0;
        flex-direction: column;

        strong,
        span {
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        strong {
          font-size: 13px;
        }

        span {
          margin-top: 3px;
          color: var(--act-text-muted);
          font-size: 11px;
        }
      }
    }

    .nav {
      border-right: 0;
      background: transparent;

      :deep(.el-menu-item) {
        height: 42px;
        margin-bottom: 5px;
        padding: 0 12px;
        border-radius: var(--act-radius-md);
        color: var(--act-text-secondary);
        font-size: 13px;

        &.is-active {
          color: var(--act-text);
          background: var(--act-primary-soft);
        }
      }
    }

    .sidebar-footer {
      display: flex;
      gap: 10px;
      align-items: center;
      justify-content: space-between;
      margin-top: auto;
      padding: 14px 4px 0;
      border-top: 1px solid var(--act-border);

      .account {
        display: flex;
        min-width: 0;
        gap: 10px;
        align-items: center;

        .avatar {
          flex: 0 0 auto;
          color: #fff;
          font-size: 12px;
          font-weight: 700;
          background: var(--act-primary);
        }

        .account-text {
          display: flex;
          min-width: 0;
          flex-direction: column;

          strong,
          span {
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
          }

          strong {
            font-size: 12px;
          }

          span {
            margin-top: 2px;
            color: var(--act-text-muted);
            font-size: 11px;
          }
        }
      }
    }
  }

  .main {
    min-width: 0;
    min-height: 0;
    flex-direction: column;

    .topbar {
      display: flex;
      flex: 0 0 auto;
      align-items: center;
      justify-content: space-between;
      padding: 0 24px;
      border-bottom: 1px solid var(--act-border);
      background: var(--act-bg);

      .topbar-left,
      .topbar-actions {
        display: flex;
        gap: 10px;
        align-items: center;
      }

      .back-button {
        flex: 0 0 auto;
      }
    }

    .content {
      min-height: 0;
      padding: 0;
      overflow: hidden;
    }
  }
}
</style>
