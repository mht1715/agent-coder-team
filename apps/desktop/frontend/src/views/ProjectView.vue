<script setup lang="ts">
import { computed, nextTick, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Bot, Cpu, RefreshCw, Send, UserRound } from 'lucide-vue-next'
import { getProject } from '../api/projects'
import {
  conversationStreamUrl,
  listMessages,
  listProjectConversations,
  sendMessage as sendMessageApi,
} from '../api/conversations'
import { getProjectProfile, scanProjectProfile } from '../api/profile'
import { tokenStorage } from '../api/token-storage'
import { connectSSE } from '../utils/sse'
import { translateApiError } from '../utils/error'
import type { Conversation, Message, Project, ProjectProfile } from '../types/api'

const route = useRoute()
const { t, locale } = useI18n()

const projectId = computed(() => String(route.params.id))
const project = ref<Project | null>(null)
const conversation = ref<Conversation | null>(null)
const messages = ref<Message[]>([])
const profile = ref<ProjectProfile | null>(null)
const loading = ref(true)
const sending = ref(false)
const scanning = ref(false)
const draft = ref('')
const messageList = ref<HTMLElement | null>(null)
let streamController: AbortController | null = null
let hasLoaded = false

async function load() {
  loading.value = true
  try {
    project.value = await getProject(projectId.value)
    const conversations = await listProjectConversations(projectId.value)
    conversation.value = conversations[0] || null
    if (conversation.value) {
      const page = await listMessages(conversation.value.id, undefined, 50)
      messages.value = [...page.items].reverse()
      await scrollToBottom()
      startRealtime()
    }
    await loadProfile()
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    loading.value = false
    hasLoaded = true
  }
}

async function loadProfile() {
  try {
    profile.value = await getProjectProfile(projectId.value)
  } catch (error) {
    const status = (error as { response?: { status?: number } }).response?.status
    if (status === 404) {
      profile.value = null
      return
    }
    ElMessage.error(translateApiError(error, t))
  }
}

function startRealtime() {
  if (!conversation.value) return
  streamController?.abort()
  streamController = new AbortController()
  const controller = streamController
  const url = conversationStreamUrl(conversation.value.id)
  const token = tokenStorage.getAccess()
  if (!token) return

  void connectSSE({
    url,
    token,
    signal: controller.signal,
    onEvent: async (event) => {
      if (event.event !== 'message.created') return
      try {
        const message = JSON.parse(event.data) as Message
        if (!messages.value.some((item) => item.id === message.id)) {
          messages.value.push(message)
          await scrollToBottom()
        }
      } catch {
        // Ignore malformed events; the next history fetch will recover.
      }
    },
    onError: () => {
      if (!controller.signal.aborted) {
        ElMessage.warning(t('chat.disconnected'))
      }
    },
  })
}

async function send() {
  if (!conversation.value || !draft.value.trim() || sending.value) return
  const content = draft.value.trim()
  draft.value = ''
  sending.value = true
  try {
    const message = await sendMessageApi(conversation.value.id, content)
    if (!messages.value.some((item) => item.id === message.id)) {
      messages.value.push(message)
    }
    await scrollToBottom()
  } catch (error) {
    draft.value = content
    ElMessage.error(translateApiError(error, t))
  } finally {
    sending.value = false
  }
}

async function scan() {
  scanning.value = true
  try {
    profile.value = await scanProjectProfile(projectId.value)
    ElMessage.success(t('chat.scanSuccess'))
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    scanning.value = false
  }
}

async function scrollToBottom() {
  await nextTick()
  const element = messageList.value
  if (element) {
    element.scrollTop = element.scrollHeight
  }
}

function senderLabel(message: Message): string {
  return t(`chat.sender.${message.sender_type}`)
}

function formatTime(value: string): string {
  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'short',
    timeStyle: 'short',
  }).format(new Date(value))
}

async function refreshOnActivate() {
  if (!hasLoaded) return
  try {
    if (conversation.value) {
      const page = await listMessages(conversation.value.id, undefined, 50)
      messages.value = [...page.items].reverse()
      startRealtime()
    }
    await loadProfile()
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  }
}

onMounted(load)
onActivated(() => {
  void refreshOnActivate()
})
onDeactivated(() => streamController?.abort())
onBeforeUnmount(() => streamController?.abort())
</script>

<template>
  <div v-loading="loading" class="project-shell">
    <section class="chat-panel">
      <header class="chat-header">
        <div>
          <h2>{{ project?.name || t('chat.title') }}</h2>
          <p>{{ project?.description || project?.working_dir }}</p>
        </div>
        <el-tag size="small" effect="plain" type="success">{{ t('chat.statusOnline') }}</el-tag>
      </header>

      <div ref="messageList" class="messages">
        <el-empty v-if="messages.length === 0" :description="t('chat.noMessages')" />
        <article
          v-for="message in messages"
          :key="message.id"
          class="message-row"
          :class="{ mine: message.sender_type === 'user' }"
        >
          <el-avatar :size="28" class="message-avatar">
            <UserRound v-if="message.sender_type === 'user'" :size="16" />
            <Bot v-else :size="16" />
          </el-avatar>
          <div class="message-body">
            <div class="message-meta">
              <span>{{ senderLabel(message) }}</span>
              <time>{{ formatTime(message.created_at) }}</time>
            </div>
            <div class="message-bubble">{{ message.content }}</div>
          </div>
        </article>
      </div>

      <footer class="composer">
        <el-input
          v-model="draft"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 6 }"
          :placeholder="t('chat.placeholder')"
          @keydown.enter.exact.prevent="send"
        />
        <div class="composer-actions">
          <el-text size="small" type="info">{{ t('chat.inputHint') }}</el-text>
          <el-button type="primary" :loading="sending" :disabled="!draft.trim()" @click="send">
            <Send :size="15" />
            <span>{{ t('chat.send') }}</span>
          </el-button>
        </div>
      </footer>
    </section>

    <aside class="profile-panel">
      <div class="profile-head">
        <div>
          <h3>{{ t('chat.profile') }}</h3>
          <p>{{ t('chat.profileDescription') }}</p>
        </div>
        <el-button text :loading="scanning" @click="scan">
          <RefreshCw :size="15" />
          <span>{{ t('chat.scanProfile') }}</span>
        </el-button>
      </div>

      <el-empty v-if="!profile" :description="t('chat.profileMissing')" :image-size="72" />
      <el-card v-else class="profile-card" shadow="never">
        <div class="profile-hero">
          <div class="language-icon"><Cpu :size="18" /></div>
          <div>
            <strong>{{ profile.primary_language || t('common.unknown') }}</strong>
            <div class="act-muted">{{ profile.package_manager || t('common.noData') }}</div>
          </div>
        </div>

        <el-descriptions class="profile-descriptions" :column="1" border size="small">
          <el-descriptions-item :label="t('chat.frameworks')">
            <div class="tag-list">
              <el-tag v-for="framework in profile.frameworks" :key="framework" size="small" effect="plain">
                {{ framework }}
              </el-tag>
              <span v-if="profile.frameworks.length === 0" class="act-muted">{{ t('common.noData') }}</span>
            </div>
          </el-descriptions-item>
          <el-descriptions-item :label="t('chat.buildCommand')">
            <code>{{ profile.build_command || t('common.noData') }}</code>
          </el-descriptions-item>
          <el-descriptions-item :label="t('chat.testCommand')">
            <code>{{ profile.test_command || t('common.noData') }}</code>
          </el-descriptions-item>
          <el-descriptions-item :label="t('projects.workingDir')">
            <code>{{ project?.working_dir || t('common.noData') }}</code>
          </el-descriptions-item>
        </el-descriptions>

        <div class="profile-foot">
          <el-text size="small" type="info">{{ t('chat.fileCount', { count: profile.file_count }) }}</el-text>
          <el-text size="small" type="info">
            {{ t('chat.updatedAt', { time: formatTime(profile.updated_at) }) }}
          </el-text>
        </div>
      </el-card>
    </aside>
  </div>
</template>

<style scoped lang="scss">
.project-shell {
  display: grid;
  height: 100%;
  min-height: 0;
  grid-template-columns: minmax(0, 1fr) 330px;

  .chat-panel {
    display: flex;
    min-width: 0;
    min-height: 0;
    flex-direction: column;
    border-right: 1px solid var(--act-border);

    .chat-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 18px 24px;
      border-bottom: 1px solid var(--act-border);

      h2 {
        margin: 0;
        font-size: 16px;
        font-weight: 600;
      }

      p {
        overflow: hidden;
        max-width: 68vw;
        margin: 4px 0 0;
        color: var(--act-text-muted);
        font-size: 12px;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }

    .messages {
      display: flex;
      min-height: 0;
      flex: 1;
      flex-direction: column;
      gap: 18px;
      padding: 20px 24px;
      overflow-y: auto;
    }

    .message-row {
      display: flex;
      max-width: 78%;
      gap: 12px;
      align-items: flex-start;

      &.mine {
        align-self: flex-end;
        flex-direction: row-reverse;

        .message-bubble {
          border-color: var(--act-primary);
          color: #fff;
          background: var(--act-primary);
        }

        .message-meta {
          justify-content: flex-end;
        }
      }

      .message-avatar {
        flex: 0 0 auto;
        color: var(--act-primary);
        background: var(--act-primary-soft);
      }

      .message-body {
        min-width: 0;

        .message-meta {
          display: flex;
          gap: 8px;
          margin-bottom: 5px;
          color: var(--act-text-muted);
          font-size: 11px;
        }

        .message-bubble {
          padding: 10px 14px;
          border: 1px solid var(--act-border);
          border-radius: var(--act-radius-md);
          background: var(--act-surface);
          font-size: 13px;
          line-height: 1.65;
          white-space: pre-wrap;
          word-break: break-word;
        }
      }
    }

    .composer {
      padding: 16px 24px 20px;
      border-top: 1px solid var(--act-border);

      .composer-actions {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
        margin-top: 10px;
      }
    }
  }

  .profile-panel {
    min-height: 0;
    padding: 20px;
    overflow-y: auto;
    background: var(--act-bg-soft);

    .profile-head {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 16px;
      margin-bottom: 18px;

      h3 {
        margin: 0;
        font-size: 14px;
        font-weight: 600;
      }

      p {
        margin: 4px 0 0;
        color: var(--act-text-muted);
        font-size: 11px;
      }
    }

    .profile-card {
      background: var(--act-surface);

      .profile-hero {
        display: flex;
        gap: 10px;
        align-items: center;
        margin-bottom: 16px;

        .language-icon {
          display: grid;
          width: 32px;
          height: 32px;
          place-items: center;
          border-radius: var(--act-radius-sm);
          color: var(--act-primary);
          background: var(--act-primary-soft);
        }

        strong {
          font-size: 13px;
        }
      }

      .profile-descriptions {
        :deep(.el-descriptions__label) {
          width: 100px;
          color: var(--act-text-muted);
          font-size: 11px;
        }

        :deep(.el-descriptions__content) {
          min-width: 0;
          font-size: 11px;
        }

        code {
          font-family: var(--act-font-mono);
          white-space: normal;
          word-break: break-all;
        }
      }

      .tag-list {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
      }

      .profile-foot {
        display: flex;
        flex-direction: column;
        gap: 6px;
        margin-top: 16px;
      }
    }
  }
}
</style>
