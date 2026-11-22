<script setup lang="ts">
import { onActivated, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Bot, ClipboardCheck, FileText, FolderKanban, ListTree } from 'lucide-vue-next'
import { getProject } from '../api/projects'
import { translateApiError } from '../utils/error'
import ProjectChatPanel from '../components/project/ProjectChatPanel.vue'
import RequirementPanel from '../components/project/RequirementPanel.vue'
import TaskBoard from '../components/project/TaskBoard.vue'
import ActivityPanel from '../components/project/ActivityPanel.vue'
import type { Project } from '../types/api'

type ProjectTab = 'chat' | 'requirements' | 'tasks' | 'activity'

const route = useRoute()
const { t } = useI18n()
const project = ref<Project | null>(null)
const loading = ref(true)
const activeTab = ref<ProjectTab>('chat')
let hasLoaded = false

async function load() {
  loading.value = true
  try {
    project.value = await getProject(String(route.params.id))
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    loading.value = false
    hasLoaded = true
  }
}

onMounted(load)
onActivated(() => {
  if (hasLoaded) void load()
})
</script>

<template>
  <div v-loading="loading" class="project-workspace">
    <header class="project-head">
      <div class="project-identity">
        <div class="project-mark"><FolderKanban :size="20" /></div>
        <div class="identity-copy">
          <div class="title-line">
            <h1>{{ project?.name || t('chat.title') }}</h1>
            <el-tag size="small" effect="plain">{{ project?.status || t('common.unknown') }}</el-tag>
          </div>
          <p>{{ project?.description || project?.working_dir }}</p>
        </div>
      </div>
      <div class="project-facts">
        <span class="fact-label">{{ t('projects.workingDir') }}</span>
        <code>{{ project?.working_dir || t('common.noData') }}</code>
      </div>
    </header>

    <el-tabs v-model="activeTab" class="workspace-tabs">
      <el-tab-pane name="chat">
        <template #label>
          <span class="tab-label"><Bot :size="15" />{{ t('agent.tabChat') }}</span>
        </template>
        <ProjectChatPanel
          v-if="project"
          :project="project"
          @open-tab="activeTab = $event"
        />
      </el-tab-pane>

      <el-tab-pane name="requirements" lazy>
        <template #label>
          <span class="tab-label"><FileText :size="15" />{{ t('agent.tabRequirements') }}</span>
        </template>
        <RequirementPanel :project-id="String(route.params.id)" @open-tab="activeTab = $event" />
      </el-tab-pane>

      <el-tab-pane name="tasks" lazy>
        <template #label>
          <span class="tab-label"><ListTree :size="15" />{{ t('agent.tabTasks') }}</span>
        </template>
        <TaskBoard :project-id="String(route.params.id)" @open-tab="activeTab = $event" />
      </el-tab-pane>

      <el-tab-pane name="activity" lazy>
        <template #label>
          <span class="tab-label"><ClipboardCheck :size="15" />{{ t('agent.tabActivity') }}</span>
        </template>
        <ActivityPanel :project-id="String(route.params.id)" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped lang="scss">
.project-workspace {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;

  .project-head {
    display: flex;
    flex: 0 0 auto;
    align-items: center;
    justify-content: space-between;
    gap: var(--act-space-6);
    padding: var(--act-space-4) var(--act-space-6);
    border-bottom: 1px solid var(--act-border);

    .project-identity {
      display: flex;
      min-width: 0;
      gap: var(--act-space-3);
      align-items: center;

      .project-mark {
        display: grid;
        width: 40px;
        height: 40px;
        flex: 0 0 auto;
        place-items: center;
        border-radius: var(--act-radius-md);
        color: var(--act-primary);
        background: var(--act-primary-soft);
      }

      .identity-copy {
        min-width: 0;

        .title-line {
          display: flex;
          gap: var(--act-space-2);
          align-items: center;

          h1 {
            overflow: hidden;
            margin: 0;
            font-size: 18px;
            font-weight: 600;
            text-overflow: ellipsis;
            white-space: nowrap;
          }
        }

        p {
          overflow: hidden;
          max-width: 640px;
          margin: 5px 0 0;
          color: var(--act-text-muted);
          font-size: 12px;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
      }
    }

    .project-facts {
      display: flex;
      min-width: 0;
      flex: 0 1 420px;
      flex-direction: column;
      align-items: flex-end;
      gap: 4px;

      .fact-label { color: var(--act-text-muted); font-size: 10px; }
      code { overflow: hidden; max-width: 100%; color: var(--act-text-secondary); font-family: var(--act-font-mono); font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
    }
  }

  .workspace-tabs {
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;

    :deep(.el-tabs__header) {
      flex: 0 0 auto;
      margin: 0;
      padding: 0 var(--act-space-6);
      border-bottom: 1px solid var(--act-border);
    }

    :deep(.el-tabs__nav-wrap::after) { display: none; }
    :deep(.el-tabs__item) { height: 46px; font-size: 12px; }

    :deep(.el-tabs__content) {
      min-height: 0;
      flex: 1;
    }

    :deep(.el-tab-pane) { height: 100%; }

    .tab-label {
      display: inline-flex;
      gap: 6px;
      align-items: center;
      vertical-align: middle;
    }
  }
}
</style>
