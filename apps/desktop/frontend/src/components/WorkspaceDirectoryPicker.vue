<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { ArrowUp, ChevronRight, Folder, FolderOpen, RefreshCw } from 'lucide-vue-next'
import { browseWorkspace } from '../api/workspace'
import { translateApiError } from '../utils/error'
import type { WorkspaceBrowseResult } from '../types/api'

const model = defineModel<string>({ default: '' })
const props = withDefaults(defineProps<{ placeholder?: string; disabled?: boolean }>(), {
  placeholder: '',
  disabled: false,
})

const { t } = useI18n()
const visible = ref(false)
const loading = ref(false)
const browser = ref<WorkspaceBrowseResult | null>(null)
const selectedRoot = ref('')

const canSelectCurrent = computed(
  () => Boolean(browser.value) && browser.value?.path !== browser.value?.root,
)

const breadcrumbs = computed(() => {
  const current = browser.value
  if (!current) return []
  const relative = current.path.slice(current.root.length)
  const parts = relative.split(/[\\/]/).filter(Boolean)
  const items = [{ label: current.root, path: current.root }]
  let path = current.root
  for (const part of parts) {
    path = `${path.replace(/[\\/]$/, '')}/${part}`
    items.push({ label: part, path })
  }
  return items
})

async function openPicker() {
  visible.value = true
  await load(model.value || undefined)
}

async function load(path?: string) {
  loading.value = true
  try {
    browser.value = await browseWorkspace(path)
    selectedRoot.value = browser.value.root
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    loading.value = false
  }
}

function navigate(path?: string) {
  if (path) void load(path)
}

function chooseRoot(root: string) {
  void load(root)
}

function selectCurrent() {
  if (!browser.value || !canSelectCurrent.value) {
    ElMessage.warning(t('directory.rootNotSelectable'))
    return
  }
  model.value = browser.value.path
  visible.value = false
}
</script>

<template>
  <div class="directory-picker">
    <el-input
      v-model="model"
      class="act-mono"
      :placeholder="props.placeholder || t('projects.workingDirPlaceholder')"
      :disabled="props.disabled"
      clearable
    >
      <template #append>
        <el-button :disabled="props.disabled" @click="openPicker">
          <FolderOpen :size="15" />
          <span>{{ t('directory.browse') }}</span>
        </el-button>
      </template>
    </el-input>

    <el-dialog v-model="visible" :title="t('directory.title')" width="720px" append-to-body>
      <div class="picker-toolbar">
        <el-button-group>
          <el-button :icon="ArrowUp" :disabled="!browser?.parent" @click="navigate(browser?.parent)">
            {{ t('directory.parent') }}
          </el-button>
          <el-button :icon="RefreshCw" :loading="loading" @click="navigate(browser?.path)">
            {{ t('common.refresh') }}
          </el-button>
        </el-button-group>
        <el-select v-model="selectedRoot" class="root-select" @change="chooseRoot">
          <el-option v-for="root in browser?.roots || []" :key="root" :label="root" :value="root" />
        </el-select>
      </div>

      <el-breadcrumb separator="/" class="picker-breadcrumb">
        <el-breadcrumb-item v-for="crumb in breadcrumbs" :key="crumb.path">
          <el-button link class="act-mono" @click="navigate(crumb.path)">{{ crumb.label }}</el-button>
        </el-breadcrumb-item>
      </el-breadcrumb>

      <el-alert :title="t('directory.selectTip')" type="info" :closable="false" show-icon />
      <el-alert
        v-if="browser?.truncated"
        :title="t('directory.truncated')"
        type="warning"
        :closable="false"
        show-icon
        class="truncated-alert"
      />

      <el-table v-loading="loading" :data="browser?.entries || []" height="320" class="picker-table">
        <el-table-column :label="t('directory.name')">
          <template #default="{ row }">
            <div class="directory-cell">
              <Folder :size="16" />
              <span class="act-mono">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column width="120" align="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="navigate(row.path)">
              <span>{{ t('directory.enter') }}</span>
              <ChevronRight :size="15" />
            </el-button>
          </template>
        </el-table-column>
        <template #empty>
          <el-empty :description="t('directory.empty')" :image-size="64" />
        </template>
      </el-table>

      <template #footer>
        <div class="dialog-footer">
          <span class="current-path act-mono">{{ browser?.path || t('common.loading') }}</span>
          <el-button @click="visible = false">{{ t('common.cancel') }}</el-button>
          <el-button type="primary" :disabled="!canSelectCurrent" @click="selectCurrent">
            {{ t('directory.selectCurrent') }}
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.directory-picker {
  width: 100%;

  .picker-toolbar {
    display: flex;
    gap: 12px;
    align-items: center;
    margin-bottom: 16px;

    .root-select {
      width: 300px;
    }
  }

  .picker-breadcrumb {
    min-height: 28px;
    margin-bottom: 12px;
  }

  .truncated-alert {
    margin-top: 12px;
  }

  :deep(.el-alert) {
    border-radius: var(--act-radius-md);
  }

  :deep(.el-table) {
    overflow: hidden;
    border-radius: var(--act-radius-md);
  }

  .picker-table {
    margin-top: 16px;
  }

  .directory-cell {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .dialog-footer {
    display: flex;
    gap: 12px;
    align-items: center;

    .current-path {
      min-width: 0;
      flex: 1;
      overflow: hidden;
      color: var(--act-text-muted);
      font-size: 13px;
      text-align: left;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }
}
</style>
