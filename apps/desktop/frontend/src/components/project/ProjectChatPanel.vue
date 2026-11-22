<script setup lang="ts">
import { computed, nextTick, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Bot, Cpu, RefreshCw, Send, UserRound } from 'lucide-vue-next'
import { listMessages, listProjectConversations, sendMessage as sendMessageApi } from '../../api/conversations'
import { getProjectProfile, scanProjectProfile } from '../../api/profile'
import { subscribeConversationStream, type ConversationStreamCleanup } from '../../utils/conversation-stream'
import { translateApiError } from '../../utils/error'
import type { Conversation, Message, MessageDelta, Project, ProjectProfile } from '../../types/api'

const props = defineProps<{ project: Project }>()
const emit = defineEmits<{ (event: 'open-tab', tab: 'requirements' | 'tasks' | 'activity'): void }>()
const { t, locale } = useI18n()

interface ChatMessage extends Message {
  streaming?: boolean
}

const conversation = ref<Conversation | null>(null)
const messages = ref<ChatMessage[]>([])
const profile = ref<ProjectProfile | null>(null)
const loading = ref(true)
const sending = ref(false)
const scanning = ref(false)
const thinking = ref(false)
const draft = ref('')
const messageList = ref<HTMLElement | null>(null)
let streamCleanup: ConversationStreamCleanup | null = null
let lastStreamErrorAt = 0
let hasLoaded = false

const agentOnline = computed(() => Boolean(conversation.value))

async function load() {
  loading.value = true
  try {
    const conversations = await listProjectConversations(props.project.id)
    conversation.value = conversations[0] || null
    if (conversation.value) {
      const page = await listMessages(conversation.value.id, undefined, 50)
      messages.value = [...page.items].reverse()
      await scrollToBottom()
      await startRealtime()
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
    profile.value = await getProjectProfile(props.project.id)
  } catch (error) {
    const status = (error as { status?: number }).status
    if (status === 404) {
      profile.value = null
      return
    }
    ElMessage.error(translateApiError(error, t))
  }
}

async function stopRealtime() {
  const cleanup = streamCleanup
  streamCleanup = null
  if (cleanup) await cleanup()
}

async function startRealtime() {
  if (!conversation.value) return
  await stopRealtime()
  const conversationID = conversation.value.id
  try {
    streamCleanup = await subscribeConversationStream(conversationID, {
      onMessage: async (message) => {
        const index = messages.value.findIndex((item) => item.id === message.id)
        if (index >= 0) messages.value[index] = message
        else messages.value.push(message)
        if (message.sender_type !== 'user') thinking.value = false
        await scrollToBottom()
      },
      onDelta: async (delta) => {
        applyDelta(delta)
        await scrollToBottom()
      },
      onThinking: (active) => {
        thinking.value = active
      },
      onEvent: (type, data) => {
        if (type === 'requirement.updated') {
          ElMessage.success(t('agent.requirementUpdated'))
        }
        if (type === 'tasks.updated' && data && typeof data === 'object') {
          const payload = data as { count?: number }
          if (payload.count) ElMessage.success(t('agent.tasksUpdated', { count: payload.count }))
        }
      },
      onError: () => {
        thinking.value = false
        const now = Date.now()
        if (now - lastStreamErrorAt > 5000) {
          lastStreamErrorAt = now
          ElMessage.warning(t('chat.disconnected'))
        }
      },
    })
  } catch (error) {
    thinking.value = false
    ElMessage.error(translateApiError(error, t))
  }
}

function applyDelta(delta: MessageDelta) {
  const index = messages.value.findIndex((item) => item.id === delta.message_id)
  if (index >= 0) {
    messages.value[index].content += delta.delta
    return
  }
  messages.value.push({
    id: delta.message_id,
    conversation_id: delta.conversation_id,
    sender_type: delta.sender_type,
    content: delta.delta,
    content_format: 'markdown',
    created_at: delta.created_at,
    streaming: true,
  })
}

async function send() {
  if (!conversation.value || !draft.value.trim() || sending.value) return
  const content = draft.value.trim()
  draft.value = ''
  sending.value = true
  thinking.value = true
  try {
    const message = await sendMessageApi(conversation.value.id, content)
    if (!messages.value.some((item) => item.id === message.id)) messages.value.push(message)
    await scrollToBottom()
  } catch (error) {
    draft.value = content
    thinking.value = false
    ElMessage.error(translateApiError(error, t))
  } finally {
    sending.value = false
  }
}

async function scan() {
  scanning.value = true
  try {
    profile.value = await scanProjectProfile(props.project.id)
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
  if (element) element.scrollTop = element.scrollHeight
}

function senderLabel(message: Message): string {
  return t(`chat.sender.${message.sender_type}`)
}

function formatTime(value: string): string {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value))
}

async function refreshOnActivate() {
  if (!hasLoaded) return
  try {
    if (conversation.value) {
      const page = await listMessages(conversation.value.id, undefined, 50)
      messages.value = [...page.items].reverse()
      void startRealtime()
    }
    await loadProfile()
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  }
}

onMounted(load)
onActivated(() => void refreshOnActivate())
onDeactivated(() => void stopRealtime())
onBeforeUnmount(() => void stopRealtime())
</script>

<template>
  <div v-loading="loading" class="chat-shell">
    <section class="chat-panel">
      <header class="chat-header">
        <div class="header-copy">
          <h2>{{ t('agent.mainAgent') }}</h2>
          <p>{{ t('agent.mainAgentDescription') }}</p>
        </div>
        <el-tag size="small" effect="plain" :type="agentOnline ? 'success' : 'info'">
          {{ agentOnline ? t('chat.statusOnline') : t('common.loading') }}
        </el-tag>
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
            <div class="message-bubble" :class="{ streaming: message.streaming }">{{ message.content }}</div>
          </div>
        </article>
        <div v-if="thinking && !messages.some((item) => item.streaming)" class="thinking-row">
          <el-avatar :size="28" class="message-avatar"><Bot :size="16" /></el-avatar>
          <div class="thinking-bubble">
            <span></span><span></span><span></span>
          </div>
        </div>
      </div>

      <footer class="composer">
        <el-input
          v-model="draft"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 6 }"
          :placeholder="t('agent.chatPlaceholder')"
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

      <el-card class="workflow-card" shadow="never">
        <h4>{{ t('agent.workflowTitle') }}</h4>
        <el-button text class="workflow-step" @click="emit('open-tab', 'requirements')">
          <span class="step-index">1</span>
          <span>{{ t('agent.workflowRequirements') }}</span>
        </el-button>
        <el-button text class="workflow-step" @click="emit('open-tab', 'tasks')">
          <span class="step-index">2</span>
          <span>{{ t('agent.workflowTasks') }}</span>
        </el-button>
        <el-button text class="workflow-step" @click="emit('open-tab', 'activity')">
          <span class="step-index">3</span>
          <span>{{ t('agent.workflowActivity') }}</span>
        </el-button>
      </el-card>

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
            <code>{{ props.project.working_dir || t('common.noData') }}</code>
          </el-descriptions-item>
        </el-descriptions>

        <div class="profile-foot">
          <el-text size="small" type="info">{{ t('chat.fileCount', { count: profile.file_count }) }}</el-text>
          <el-text size="small" type="info">{{ t('chat.updatedAt', { time: formatTime(profile.updated_at) }) }}</el-text>
        </div>
      </el-card>
    </aside>
  </div>
</template>

<style scoped lang="scss">
.chat-shell {
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
      flex: 0 0 auto;
      align-items: center;
      justify-content: space-between;
      gap: var(--act-space-4);
      padding: 16px var(--act-space-6);
      border-bottom: 1px solid var(--act-border);

      h2 {
        margin: 0;
        font-size: 15px;
        font-weight: 600;
      }

      p {
        margin: 4px 0 0;
        color: var(--act-text-muted);
        font-size: 11px;
      }
    }

    .messages {
      display: flex;
      min-height: 0;
      flex: 1;
      flex-direction: column;
      gap: var(--act-space-4);
      padding: var(--act-space-5) var(--act-space-6);
      overflow-y: auto;
    }

    .message-row {
      display: flex;
      max-width: 82%;
      gap: var(--act-space-3);
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
          gap: var(--act-space-2);
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

          &.streaming::after {
            display: inline-block;
            width: 6px;
            height: 14px;
            margin-left: 3px;
            vertical-align: -2px;
            background: var(--act-primary);
            content: '';
            animation: blink 0.9s steps(2, start) infinite;
          }
        }
      }
    }

    .thinking-row {
      display: flex;
      gap: var(--act-space-3);
      align-items: center;

      .message-avatar {
        color: var(--act-primary);
        background: var(--act-primary-soft);
      }

      .thinking-bubble {
        display: flex;
        gap: 4px;
        padding: 12px 14px;
        border: 1px solid var(--act-border);
        border-radius: var(--act-radius-md);

        span {
          width: 5px;
          height: 5px;
          border-radius: 50%;
          background: var(--act-text-muted);
          animation: thinking 1.1s infinite ease-in-out;
        }

        span:nth-child(2) { animation-delay: 0.15s; }
        span:nth-child(3) { animation-delay: 0.3s; }
      }
    }

    .composer {
      flex: 0 0 auto;
      padding: var(--act-space-4) var(--act-space-6) var(--act-space-5);
      border-top: 1px solid var(--act-border);

      .composer-actions {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: var(--act-space-4);
        margin-top: var(--act-space-2);
      }
    }
  }

  .profile-panel {
    min-height: 0;
    padding: var(--act-space-5);
    overflow-y: auto;
    background: var(--act-bg-soft);

    .profile-head {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: var(--act-space-4);
      margin-bottom: var(--act-space-4);

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

    .workflow-card {
      margin-bottom: var(--act-space-4);
      background: var(--act-surface);

      h4 {
        margin: 0 0 var(--act-space-2);
        font-size: 12px;
        font-weight: 600;
      }

      .workflow-step {
        display: flex;
        width: 100%;
        height: 36px;
        justify-content: flex-start;
        gap: var(--act-space-2);
        padding: 0 var(--act-space-2);
        color: var(--act-text-secondary);

        .step-index {
          display: grid;
          width: 20px;
          height: 20px;
          place-items: center;
          border-radius: 50%;
          color: var(--act-primary);
          background: var(--act-primary-soft);
          font-size: 10px;
          font-weight: 700;
        }
      }
    }

    .profile-card {
      background: var(--act-surface);

      .profile-hero {
        display: flex;
        gap: var(--act-space-2);
        align-items: center;
        margin-bottom: var(--act-space-4);

        .language-icon {
          display: grid;
          width: 32px;
          height: 32px;
          place-items: center;
          border-radius: var(--act-radius-sm);
          color: var(--act-primary);
          background: var(--act-primary-soft);
        }

        strong { font-size: 13px; }
      }

      .profile-descriptions {
        :deep(.el-descriptions__label) { width: 100px; color: var(--act-text-muted); font-size: 11px; }
        :deep(.el-descriptions__content) { min-width: 0; font-size: 11px; }
        code { font-family: var(--act-font-mono); white-space: normal; word-break: break-all; }
      }

      .tag-list { display: flex; flex-wrap: wrap; gap: 6px; }
      .profile-foot { display: flex; flex-direction: column; gap: 6px; margin-top: var(--act-space-4); }
    }
  }
}

@keyframes blink {
  50% { opacity: 0; }
}

@keyframes thinking {
  0%, 80%, 100% { opacity: 0.3; transform: translateY(0); }
  40% { opacity: 1; transform: translateY(-2px); }
}
</style>
