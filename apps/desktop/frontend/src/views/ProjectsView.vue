<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { ChevronRight, FolderOpen, Plus } from 'lucide-vue-next'
import WorkspaceDirectoryPicker from '../components/WorkspaceDirectoryPicker.vue'
import { createProject, listProjects } from '../api/projects'
import { translateApiError } from '../utils/error'
import type { Project } from '../types/api'

const router = useRouter()
const { t, locale } = useI18n()

const projects = ref<Project[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const creating = ref(false)
const formRef = ref<FormInstance>()
const form = reactive({
  name: '',
  description: '',
  working_dir: '/workspaces/',
  project_mode: 'new' as 'new' | 'existing',
  output_language: 'zh-CN',
})

const rules = computed<FormRules>(() => ({
  name: [
    { required: true, message: t('validation.nameRequired'), trigger: 'blur' },
    { max: 128, message: t('validation.nameLength'), trigger: 'blur' },
  ],
  working_dir: [
    { required: true, message: t('validation.workingDirRequired'), trigger: 'change' },
  ],
}))

async function load() {
  loading.value = true
  try {
    projects.value = await listProjects()
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    loading.value = false
  }
}

function openCreate() {
  form.name = ''
  form.description = ''
  form.working_dir = '/workspaces/'
  form.project_mode = 'new'
  form.output_language = locale.value === 'en-US' ? 'en-US' : 'zh-CN'
  dialogVisible.value = true
  formRef.value?.clearValidate()
}

async function submitCreate() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  creating.value = true
  try {
    const project = await createProject({
      name: form.name.trim(),
      description: form.description.trim(),
      working_dir: form.working_dir.trim(),
      project_mode: form.project_mode,
      output_language: form.output_language,
    })
    ElMessage.success(t('projects.created'))
    dialogVisible.value = false
    if (!projects.value.some((item) => item.id === project.id)) {
      projects.value.unshift(project)
    }
    await router.push(`/projects/${project.id}`)
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    creating.value = false
  }
}

function openProject(project: Project) {
  router.push(`/projects/${project.id}`)
}

function projectModeLabel(mode: Project['project_mode']) {
  return mode === 'new' ? t('projects.modeNew') : t('projects.modeExisting')
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium' }).format(new Date(value))
}

onMounted(load)
</script>

<template>
  <div class="projects-page">
    <div class="page-head">
      <div class="page-title">
        <h2>{{ t('projects.title') }}</h2>
        <p>{{ t('projects.subtitle') }}</p>
      </div>
      <el-button type="primary" @click="openCreate">
        <Plus :size="16" />
        <span>{{ t('projects.create') }}</span>
      </el-button>
    </div>

    <div class="page-body">
      <div v-loading="loading" class="project-grid">
      <el-empty v-if="!loading && projects.length === 0" :description="t('projects.empty')" />
      <el-card
        v-for="project in projects"
        :key="project.id"
        class="project-card"
        shadow="hover"
        @click="openProject(project)"
      >
        <div class="project-top">
          <div class="project-icon"><FolderOpen :size="18" /></div>
          <el-tag size="small" effect="plain">{{ projectModeLabel(project.project_mode) }}</el-tag>
        </div>
        <h3>{{ project.name }}</h3>
        <p>{{ project.description || project.working_dir }}</p>
        <div class="project-meta">
          <el-tooltip :content="project.working_dir" placement="top">
            <span class="act-mono">{{ project.working_dir }}</span>
          </el-tooltip>
          <ChevronRight :size="16" />
        </div>
        <span class="project-date">{{ formatDate(project.created_at) }}</span>
      </el-card>
      </div>
    </div>

    <el-dialog v-model="dialogVisible" :title="t('projects.create')" width="640px" append-to-body>
      <el-form ref="formRef" :model="form" :rules="rules" label-position="top">
        <el-form-item :label="t('projects.name')" prop="name">
          <el-input
            v-model="form.name"
            :placeholder="t('projects.namePlaceholder')"
            maxlength="128"
            show-word-limit
            autofocus
          />
          <el-text size="small" type="info">{{ t('projects.nameHelp') }}</el-text>
        </el-form-item>

        <el-form-item :label="t('projects.description')" prop="description">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="3"
            maxlength="500"
            show-word-limit
            :placeholder="t('projects.descriptionPlaceholder')"
          />
          <el-text size="small" type="info">{{ t('projects.descriptionHelp') }}</el-text>
        </el-form-item>

        <el-form-item :label="t('projects.workingDir')" prop="working_dir">
          <WorkspaceDirectoryPicker
            v-model="form.working_dir"
            :placeholder="t('projects.workingDirPlaceholder')"
          />
          <el-text size="small" type="info">{{ t('projects.workingDirTip') }}</el-text>
        </el-form-item>

        <el-form-item :label="t('projects.mode')">
          <el-radio-group v-model="form.project_mode">
            <el-radio-button value="new">{{ t('projects.newProject') }}</el-radio-button>
            <el-radio-button value="existing">{{ t('projects.existingProject') }}</el-radio-button>
          </el-radio-group>
          <el-text size="small" type="info">{{ t('projects.modeHelp') }}</el-text>
        </el-form-item>

        <el-form-item :label="t('projects.outputLanguage')">
          <el-select v-model="form.output_language" class="full-width">
            <el-option :label="t('projects.languageZhCN')" value="zh-CN" />
            <el-option :label="t('projects.languageEnUS')" value="en-US" />
          </el-select>
          <el-text size="small" type="info">{{ t('projects.outputLanguageHelp') }}</el-text>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="creating" @click="submitCreate">
          {{ t('common.create') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.projects-page {
  display: flex;
  height: 100%;
  flex-direction: column;
  padding: 24px 28px 0;
  overflow: hidden;

  .page-head {
    display: flex;
    flex: 0 0 auto;
    gap: 20px;
    align-items: flex-end;
    justify-content: space-between;
    margin-bottom: 22px;

    .page-title {
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
  }

  .page-body {
    min-height: 0;
    flex: 1;
    padding: 2px 6px 28px 2px;
    overflow-y: auto;
  }

  .project-grid {
    display: grid;
    min-height: 160px;
    gap: 18px;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  }

  .project-card {
    border-radius: var(--act-radius-lg);
    cursor: pointer;
    transition: border-color 0.15s ease, transform 0.15s ease;

    &:hover {
      border-color: var(--act-border-strong);
      transform: translateY(-1px);
    }

    :deep(.el-card__body) {
      display: flex;
      min-height: 180px;
      flex-direction: column;
      padding: 20px;
    }

    .project-top {
      display: flex;
      align-items: center;
      justify-content: space-between;

      .project-icon {
        display: grid;
        width: 38px;
        height: 38px;
        place-items: center;
        border-radius: var(--act-radius-md);
        color: var(--act-primary);
        background: var(--act-primary-soft);
      }
    }

    h3 {
      margin: 18px 0 8px;
      font-size: 15px;
      font-weight: 600;
    }

    p {
      min-height: 42px;
      margin: 0;
      color: var(--act-text-muted);
      font-size: 12px;
      line-height: 1.6;
    }

    .project-meta {
      display: flex;
      gap: 10px;
      align-items: center;
      justify-content: space-between;
      margin-top: 18px;
      padding-top: 14px;
      color: var(--act-text-muted);

      span {
        overflow: hidden;
        font-size: 11px;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }

    .project-date {
      display: block;
      margin-top: 9px;
      color: var(--act-text-muted);
      font-size: 11px;
      text-align: left;
    }
  }

  :deep(.el-form-item) {
    margin-bottom: 22px;
  }

  :deep(.el-form-item .el-text) {
    margin-top: 8px;
    line-height: 1.55;
  }

  .full-width {
    width: 100%;
  }
}
</style>