<script setup lang="ts">
import { computed, onActivated, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CheckCircle2, FilePenLine, RefreshCw, Save } from 'lucide-vue-next'
import { approveRequirement, generateRequirement, getRequirement, updateRequirement } from '../../api/agent'
import { translateApiError } from '../../utils/error'
import type { RequirementDocument } from '../../types/api'

const props = defineProps<{ projectId: string }>()
const emit = defineEmits<{ (event: 'open-tab', tab: 'tasks' | 'activity'): void }>()
const { t, locale } = useI18n()

const requirement = ref<RequirementDocument | null>(null)
const loading = ref(false)
const generating = ref(false)
const saving = ref(false)
const approving = ref(false)
const editing = ref(false)
const form = reactive({ title: '', content_md: '' })

const approved = computed(() => requirement.value?.status === 'approved')

function applyRequirement(value: RequirementDocument) {
  requirement.value = value
  form.title = value.title
  form.content_md = value.content_md
  editing.value = false
}

async function load(silent = false) {
  if (!silent) loading.value = true
  try {
    applyRequirement(await getRequirement(props.projectId))
  } catch (error) {
    const status = (error as { status?: number }).status
    if (status === 404) {
      requirement.value = null
      form.title = ''
      form.content_md = ''
      return
    }
    ElMessage.error(translateApiError(error, t))
  } finally {
    if (!silent) loading.value = false
  }
}

async function generate() {
  if (requirement.value) {
    const confirmed = await ElMessageBox.confirm(
      t('agent.regenerateRequirementConfirm'),
      t('common.confirm'),
      { type: 'warning' },
    ).catch(() => false)
    if (!confirmed) return
  }
  generating.value = true
  try {
    applyRequirement(await generateRequirement(props.projectId))
    ElMessage.success(t('agent.requirementGenerated'))
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    generating.value = false
  }
}

async function save() {
  if (!form.title.trim() || !form.content_md.trim()) {
    ElMessage.warning(t('agent.requirementRequired'))
    return
  }
  saving.value = true
  try {
    applyRequirement(await updateRequirement(props.projectId, form.title, form.content_md))
    ElMessage.success(t('agent.requirementSaved'))
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    saving.value = false
  }
}

async function approve() {
  approving.value = true
  try {
    applyRequirement(await approveRequirement(props.projectId))
    ElMessage.success(t('agent.requirementApproved'))
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    approving.value = false
  }
}

function formatTime(value: string) {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}

onMounted(() => void load())
onActivated(() => void load(true))
</script>

<template>
  <div v-loading="loading" class="requirement-panel">
    <header class="panel-head">
      <div>
        <h3>{{ t('agent.requirementTitle') }}</h3>
        <p>{{ t('agent.requirementDescription') }}</p>
      </div>
      <div class="head-actions">
        <el-button :loading="generating" @click="generate">
          <RefreshCw :size="15" />
          <span>{{ requirement ? t('agent.regenerateRequirement') : t('agent.generateRequirement') }}</span>
        </el-button>
        <el-button v-if="requirement" :disabled="approved" @click="approve" type="primary" :loading="approving">
          <CheckCircle2 :size="15" />
          <span>{{ approved ? t('agent.requirementIsApproved') : t('agent.approveRequirement') }}</span>
        </el-button>
      </div>
    </header>

    <div v-if="!requirement" class="empty-state">
      <el-empty :description="t('agent.requirementEmpty')">
        <el-button type="primary" :loading="generating" @click="generate">
          <FilePenLine :size="16" />
          <span>{{ t('agent.generateRequirement') }}</span>
        </el-button>
      </el-empty>
    </div>

    <div v-else class="requirement-body">
      <div class="document-meta">
        <div class="meta-tags">
          <el-tag :type="approved ? 'success' : 'warning'" effect="plain" size="small">
            {{ approved ? t('agent.requirementApproved') : t('agent.requirementDraft') }}
          </el-tag>
          <el-tag type="info" effect="plain" size="small">v{{ requirement.version }}</el-tag>
          <el-tag effect="plain" size="small">{{ requirement.generated_by }}</el-tag>
        </div>
        <el-text size="small" type="info">{{ t('agent.updatedAt', { time: formatTime(requirement.updated_at) }) }}</el-text>
      </div>

      <div class="document-toolbar">
        <el-input
          v-if="editing"
          v-model="form.title"
          class="title-input"
          maxlength="255"
          :placeholder="t('agent.requirementName')"
        />
        <h2 v-else>{{ requirement.title }}</h2>
        <div>
          <el-button v-if="!editing" text @click="editing = true">
            <FilePenLine :size="15" />
            <span>{{ t('agent.editRequirement') }}</span>
          </el-button>
          <el-button v-if="editing" :loading="saving" type="primary" @click="save">
            <Save :size="15" />
            <span>{{ t('common.save') }}</span>
          </el-button>
          <el-button v-if="editing" @click="load()">{{ t('common.cancel') }}</el-button>
        </div>
      </div>

      <div class="document-editor">
        <el-input
          v-if="editing"
          v-model="form.content_md"
          type="textarea"
          class="markdown-input"
          :autosize="{ minRows: 20, maxRows: 40 }"
        />
        <pre v-else class="markdown-preview">{{ requirement.content_md }}</pre>
      </div>

      <footer class="document-footer">
        <el-text size="small" type="info">{{ t('agent.requirementEditHint') }}</el-text>
        <div class="footer-actions">
          <el-button @click="emit('open-tab', 'tasks')">{{ t('agent.goToTasks') }}</el-button>
          <el-button @click="emit('open-tab', 'activity')">{{ t('agent.goToActivity') }}</el-button>
        </div>
      </footer>
    </div>
  </div>
</template>

<style scoped lang="scss">
.requirement-panel {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  padding: var(--act-space-5) var(--act-space-6);

  .panel-head {
    display: flex;
    flex: 0 0 auto;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--act-space-5);
    margin-bottom: var(--act-space-4);

    h3 { margin: 0; font-size: 16px; font-weight: 600; }
    p { margin: 5px 0 0; color: var(--act-text-muted); font-size: 12px; }

    .head-actions { display: flex; gap: var(--act-space-2); flex: 0 0 auto; }
  }

  .empty-state {
    display: grid;
    min-height: 0;
    flex: 1;
    place-items: center;
  }

  .requirement-body {
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
    padding: var(--act-space-5);
    border: 1px solid var(--act-border);
    border-radius: var(--act-radius-lg);
    background: var(--act-surface);

    .document-meta {
      display: flex;
      flex: 0 0 auto;
      align-items: center;
      justify-content: space-between;
      gap: var(--act-space-4);
      padding-bottom: var(--act-space-3);
      border-bottom: 1px solid var(--act-border);

      .meta-tags { display: flex; gap: var(--act-space-2); flex-wrap: wrap; }
    }

    .document-toolbar {
      display: flex;
      flex: 0 0 auto;
      align-items: center;
      justify-content: space-between;
      gap: var(--act-space-4);
      padding: var(--act-space-4) 0 var(--act-space-3);

      h2 { min-width: 0; margin: 0; overflow: hidden; font-size: 18px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
      .title-input { max-width: 720px; }
    }

    .document-editor {
      min-height: 0;
      flex: 1;
      overflow-y: auto;

      .markdown-input {
        :deep(.el-textarea__inner) {
          min-height: 100% !important;
          padding: var(--act-space-4);
          font-family: var(--act-font-mono);
          font-size: 12px;
          line-height: 1.7;
        }
      }

      .markdown-preview {
        min-height: 100%;
        margin: 0;
        padding: var(--act-space-4);
        border: 1px solid var(--act-border);
        border-radius: var(--act-radius-sm);
        background: var(--act-bg-soft);
        font-family: var(--act-font-mono);
        font-size: 12px;
        line-height: 1.75;
        white-space: pre-wrap;
        word-break: break-word;
      }
    }

    .document-footer {
      display: flex;
      flex: 0 0 auto;
      align-items: center;
      justify-content: space-between;
      gap: var(--act-space-4);
      padding-top: var(--act-space-3);
      border-top: 1px solid var(--act-border);

      .footer-actions { display: flex; gap: var(--act-space-2); }
    }
  }
}
</style>
