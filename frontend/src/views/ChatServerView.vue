<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import ChatChannelList from '@/components/features/ChatChannelList.vue'
import ChatMessageList from '@/components/features/ChatMessageList.vue'
import ChatComposer from '@/components/features/ChatComposer.vue'
import ChatIcon from '@/components/features/ChatIcon.vue'
import TypingIndicator from '@/components/features/TypingIndicator.vue'
import ReportDialog from '@/components/features/ReportDialog.vue'
import {
  createChannel,
  deleteMessage,
  getPresence,
  listChannels,
  listMessages,
  listServers,
  reactToMessage,
  reportMessage,
  sendMessage,
} from '@/api/chat'
import { useChatSocket } from '@/composables/useChatSocket'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { errorMessage, isForbidden, isNotFound, plural } from '@/components/features/format'
import {
  isVisibleChannel,
  mergeMessages,
  normalizePresence,
  normalizeSocketMessages,
  normalizeTyping,
} from '@/components/features/chatMessageUtils'
import type { ChatChannel, ChatMessage, ChatServer } from '@/types'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()

const {
  messages: socketMessages,
  typing: socketTyping,
  presence: socketPresence,
  lastEvent,
  connect,
  disconnect,
  subscribe,
  sendTyping,
} = useChatSocket()

const serverId = computed(() => Number(route.params.id))
const scope = computed(() => ({ channelId: activeChannelId.value ?? undefined }))

const server = ref<ChatServer | null>(null)
const channels = ref<ChatChannel[]>([])
const messages = ref<ChatMessage[]>([])
const presence = ref<Record<number, string>>({})

const loadingServer = ref(true)
const loadingChannels = ref(true)
const loadingMessages = ref(false)
const loadingMore = ref(false)
const serverError = ref('')
const channelsError = ref('')
const messagesError = ref('')
const forbidden = ref(false)
const notFound = ref(false)
const nextCursor = ref('')
const cursor = ref('')
const activeChannelId = ref<number | null>(null)
const replyTo = ref<ChatMessage | null>(null)
const reportTarget = ref<ChatMessage | null>(null)
const actionBusy = ref(false)
const online = ref(typeof navigator === 'undefined' ? true : navigator.onLine)
const mobilePane = ref<'channels' | 'messages'>('channels')
const channelFormOpen = ref(false)
const channelName = ref('')
const channelDescription = ref('')
const creatingChannel = ref(false)
const channelError = ref('')
const composer = ref<InstanceType<typeof ChatComposer> | null>(null)

const canManageChannels = computed(() => server.value?.role === 'owner' || server.value?.role === 'moderator')
const activeChannel = computed(() => channels.value.find((channel) => channel.id === activeChannelId.value) || null)
const authors = computed(() => {
  const people = messages.value.map((message) => message.author).filter(Boolean)
  if (server.value?.owner) people.unshift(server.value.owner)
  return people as NonNullable<ChatMessage['author']>[]
})

const typingNames = computed(() => normalizeTyping(socketTyping.value, scope.value, authors.value))
const channelPresence = computed(() => presence.value)

function onOnline(): void {
  online.value = true
}

function onOffline(): void {
  online.value = false
}

async function loadServer(): Promise<void> {
  loadingServer.value = true
  serverError.value = ''
  forbidden.value = false
  notFound.value = false
  try {
    const servers = await listServers()
    const found = servers.find((item) => item.id === serverId.value) || null
    server.value = found
    if (!found) notFound.value = true
  } catch (cause) {
    server.value = null
    if (isForbidden(cause)) forbidden.value = true
    else if (isNotFound(cause)) notFound.value = true
    else serverError.value = errorMessage(cause, 'We could not load this server. Try again.')
  } finally {
    loadingServer.value = false
  }
}

async function loadChannels(): Promise<void> {
  loadingChannels.value = true
  channelsError.value = ''
  try {
    const list = await listChannels(serverId.value)
    channels.value = list.filter(isVisibleChannel)
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else channelsError.value = errorMessage(cause, 'We could not load channels. Try again.')
    channels.value = (server.value?.channels || []).filter(isVisibleChannel)
  } finally {
    loadingChannels.value = false
    if (!activeChannelId.value && channels.value.length) {
      const requested = Number(route.query.channel)
      activeChannelId.value = channels.value.some((channel) => channel.id === requested)
        ? requested
        : channels.value[0]!.id
    }
  }
}

async function loadPresence(): Promise<void> {
  const ids = [...new Set(authors.value.map((person) => person.id))].filter((id) => Number.isFinite(id))
  if (!ids.length) return
  try {
    const initial = await getPresence(ids)
    presence.value = { ...initial, ...presence.value }
  } catch {
    // Presence is optional; the message list still works without it.
  }
}

async function loadMessages(): Promise<void> {
  const channelId = activeChannelId.value
  if (!channelId) return
  loadingMessages.value = true
  messagesError.value = ''
  cursor.value = ''
  nextCursor.value = ''
  messages.value = []
  try {
    const page = await listMessages({ channelId, limit: 50 })
    messages.value = mergeMessages([], page.items || [])
    nextCursor.value = page.next_cursor || ''
    cursor.value = page.next_cursor || ''
    void loadPresence()
  } catch (cause) {
    messages.value = []
    if (isForbidden(cause)) forbidden.value = true
    else messagesError.value = errorMessage(cause, 'We could not load messages. Try again.')
  } finally {
    loadingMessages.value = false
  }
}

async function loadMore(): Promise<void> {
  const channelId = activeChannelId.value
  if (!channelId || !cursor.value || loadingMore.value) return
  loadingMore.value = true
  try {
    const page = await listMessages({ channelId, limit: 50, cursor: cursor.value })
    messages.value = mergeMessages(page.items || [], messages.value)
    cursor.value = page.next_cursor || ''
  } catch (cause) {
    ui.pushToast({ title: 'Could not load earlier messages', message: errorMessage(cause), tone: 'error' })
  } finally {
    loadingMore.value = false
  }
}

function replaceMessage(updated: ChatMessage): void {
  messages.value = messages.value.map((message) => (message.id === updated.id ? updated : message))
}

function selectChannel(id: number): void {
  if (activeChannelId.value === id) return
  activeChannelId.value = id
  replyTo.value = null
  mobilePane.value = 'messages'
  void router.replace({ query: { ...route.query, channel: String(id) } })
}

async function send(body: string): Promise<void> {
  const channelId = activeChannelId.value
  if (!channelId) return
  try {
    const message = await sendMessage({ channelId }, body, replyTo.value?.id)
    messages.value = mergeMessages(messages.value, [message])
    replyTo.value = null
  } catch (cause) {
    ui.pushToast({ title: 'Message not sent', message: errorMessage(cause), tone: 'error' })
  }
}

function onTyping(isTyping: boolean): void {
  const channelId = activeChannelId.value
  if (channelId) void sendTyping({ channelId }, isTyping)
}

async function onReact(message: ChatMessage, code: string): Promise<void> {
  if (actionBusy.value) return
  actionBusy.value = true
  try {
    replaceMessage(await reactToMessage(message.id, code))
  } catch (cause) {
    ui.pushToast({ title: 'Reaction not saved', message: errorMessage(cause), tone: 'error' })
  } finally {
    actionBusy.value = false
  }
}

async function onDelete(message: ChatMessage): Promise<void> {
  if (actionBusy.value) return
  actionBusy.value = true
  try {
    await deleteMessage(message.id)
    replaceMessage({ ...message, deleted: true, reactions: [] })
    ui.pushToast({ title: 'Message removed', tone: 'success' })
  } catch (cause) {
    ui.pushToast({ title: 'Message not removed', message: errorMessage(cause), tone: 'error' })
  } finally {
    actionBusy.value = false
  }
}

async function submitReport(input: { reason: string; details?: string }): Promise<void> {
  if (!reportTarget.value) return
  await reportMessage(reportTarget.value.id, input.reason)
}

async function createChannelSubmit(): Promise<void> {
  if (!channelName.value.trim() || creatingChannel.value) return
  creatingChannel.value = true
  channelError.value = ''
  try {
    const channel = await createChannel(serverId.value, {
      name: channelName.value.trim(),
      description: channelDescription.value.trim(),
    })
    channels.value = [...channels.value.filter((item) => item.id !== channel.id), channel]
    channelName.value = ''
    channelDescription.value = ''
    channelFormOpen.value = false
    selectChannel(channel.id)
  } catch (cause) {
    channelError.value = errorMessage(cause, 'We could not create this channel. Try again.')
  } finally {
    creatingChannel.value = false
  }
}

function focusComposer(): void {
  composer.value?.focus()
}

watch(socketMessages, (value) => {
  const incoming = normalizeSocketMessages(value, scope.value)
  if (incoming.length) messages.value = mergeMessages(messages.value, incoming)
})

watch(socketPresence, (value) => {
  const next = normalizePresence(value)
  if (Object.keys(next).length) presence.value = { ...presence.value, ...next }
})

watch(lastEvent, (event) => {
  if (event?.type !== 'message.deleted') return
  const payload = event.payload || {}
  const id = typeof payload.id === 'number' ? payload.id : payload.message_id
  if (typeof id === 'number') {
    messages.value = messages.value.map((message) =>
      message.id === id ? { ...message, deleted: true, reactions: [] } : message,
    )
  }
})

watch(activeChannelId, (id) => {
  if (!id) return
  subscribe(scope.value)
  void loadMessages()
})

watch(channels, (list) => {
  if (activeChannelId.value && !list.some((channel) => channel.id === activeChannelId.value)) {
    activeChannelId.value = list[0]?.id ?? null
  }
})

onMounted(() => {
  connect()
  subscribe(scope.value)
  void Promise.all([loadServer(), loadChannels()])
  window.addEventListener('online', onOnline)
  window.addEventListener('offline', onOffline)
})

onBeforeUnmount(() => {
  disconnect()
  window.removeEventListener('online', onOnline)
  window.removeEventListener('offline', onOffline)
})
</script>

<template>
  <div class="chat-server" :class="{ 'chat-server--show-messages': mobilePane === 'messages' }">
    <header class="chat-server__head">
      <button
        type="button"
        class="chat-server__back"
        :class="{ 'chat-server__back--visible': mobilePane === 'messages' }"
        aria-label="Back to channels"
        @click="mobilePane = 'channels'"
      >
        <ChatIcon name="back" :size="18" />
      </button>
      <div class="chat-server__identity">
        <h1 class="chat-server__title">{{ server?.name || 'Server' }}</h1>
        <p v-if="server?.description" class="chat-server__description">{{ server.description }}</p>
      </div>
      <span v-if="server" class="chat-server__members">
        <ChatIcon name="members" :size="16" />
        <span class="mono">{{ plural(server.member_count ?? 0, 'member') }}</span>
      </span>
    </header>

    <p v-if="!online" class="chat-server__offline" role="status">
      <ChatIcon name="offline" :size="16" />
      Reconnecting. Messages will catch up when the connection returns.
    </p>

    <SkeletonList v-if="loadingServer" :rows="3" />

    <EmptyState
      v-else-if="notFound"
      title="This server is not available"
      description="It may have been removed, or the link may be wrong."
      action-label="Back to chat"
      @action="router.push('/chat')"
    />

    <EmptyState
      v-else-if="forbidden"
      title="You do not have access to this server"
      description="Ask a server owner or moderator to add you."
      action-label="Back to chat"
      @action="router.push('/chat')"
    />

    <EmptyState
      v-else-if="serverError"
      title="We could not load this server"
      :description="serverError"
      action-label="Try again"
      @action="loadServer"
    />

    <div v-else class="chat-server__grid">
      <aside class="chat-server__sidebar" aria-label="Server channels">
        <ChatChannelList
          :channels="channels"
          :loading="loadingChannels"
          :error="channelsError"
          :forbidden="forbidden"
          :active-id="activeChannelId ?? undefined"
          :can-create="canManageChannels"
          @select="selectChannel"
          @create="channelFormOpen = !channelFormOpen"
          @retry="loadChannels"
        />

        <form v-if="channelFormOpen && canManageChannels" class="chat-server__channel-form" @submit.prevent="createChannelSubmit">
          <BaseInput v-model="channelName" label="Channel name" placeholder="e.g. general" />
          <BaseInput v-model="channelDescription" label="Description" placeholder="What belongs here?" />
          <p v-if="channelError" class="chat-server__form-error" role="alert">{{ channelError }}</p>
          <div class="chat-server__form-actions">
            <BaseButton variant="ghost" size="sm" @click="channelFormOpen = false">Cancel</BaseButton>
            <BaseButton type="submit" variant="primary" size="sm" :loading="creatingChannel" :disabled="!channelName.trim()">
              Create channel
            </BaseButton>
          </div>
        </form>
      </aside>

      <section class="chat-server__pane" aria-label="Messages">
        <header class="chat-server__pane-head">
          <div class="chat-server__channel">
            <ChatIcon name="hash" :size="16" />
            <h2 class="chat-server__channel-name">{{ activeChannel?.name || 'No channel selected' }}</h2>
          </div>
          <p v-if="activeChannel?.description" class="chat-server__channel-description">{{ activeChannel.description }}</p>
        </header>

        <template v-if="!activeChannel">
          <EmptyState
            title="Pick a channel"
            description="Choose a channel on the left to read and join the conversation."
          />
        </template>

        <template v-else>
          <ChatMessageList
            :messages="messages"
            :current-user-id="auth.user?.id"
            :presence="channelPresence"
            :loading="loadingMessages"
            :error="messagesError"
            :forbidden="forbidden"
            @react="onReact"
            @reply="replyTo = $event"
            @report="reportTarget = $event"
            @delete="onDelete"
            @focus-composer="focusComposer"
            @retry="loadMessages"
          />

          <div v-if="cursor" class="chat-server__more">
            <BaseButton variant="ghost" size="sm" :loading="loadingMore" @click="loadMore">Load earlier messages</BaseButton>
          </div>

          <div class="chat-server__compose">
            <TypingIndicator :names="typingNames" />
            <ChatComposer
              ref="composer"
              :replying-to="replyTo ? (replyTo.author?.display_name || replyTo.author?.username || 'member') : null"
              :placeholder="`Message #${activeChannel.name}`"
              @send="send"
              @typing="onTyping"
              @cancel-reply="replyTo = null"
            />
          </div>
        </template>
      </section>
    </div>

    <ReportDialog
      v-if="reportTarget"
      target-type="message"
      :target-id="reportTarget.id"
      :submitter="submitReport"
      @close="reportTarget = null"
      @submitted="reportTarget = null"
    />
  </div>
</template>

<style scoped>
.chat-server {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
}

.chat-server__head {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
}

.chat-server__back {
  display: none;
  width: 44px;
  height: 44px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
}

.chat-server__identity {
  min-width: 0;
  flex: 1;
}

.chat-server__title {
  font-size: var(--text-2xl);
}

.chat-server__description {
  margin-top: var(--space-1);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.chat-server__members {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding-top: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.chat-server__offline {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.chat-server__grid {
  display: grid;
  grid-template-columns: minmax(0, 176px) minmax(0, 1fr);
  gap: var(--space-4);
  min-width: 0;
}

.chat-server__sidebar {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  min-width: 0;
  max-height: min(72vh, 640px);
  padding: var(--space-3);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-850);
}

.chat-server__channel-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding-top: var(--space-3);
  border-top: 1px solid var(--border-hairline);
}

.chat-server__form-error {
  color: var(--alert);
  font-size: var(--text-xs);
}

.chat-server__form-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-1);
}

.chat-server__pane {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: min(72vh, 640px);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-850);
}

.chat-server__pane-head {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--border-hairline);
}

.chat-server__channel {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-secondary);
}

.chat-server__channel-name {
  color: var(--text-primary);
  font-family: var(--font-body);
  font-size: var(--text-lg);
  font-weight: 600;
}

.chat-server__channel-description {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.chat-server__more {
  display: flex;
  justify-content: center;
  padding: var(--space-2);
}

.chat-server__compose {
  margin-top: auto;
}

@media (max-width: 767px) {
  .chat-server__back--visible {
    display: inline-flex;
  }

  .chat-server__members {
    font-size: var(--text-xs);
  }

  .chat-server__grid {
    display: block;
  }

  .chat-server__sidebar {
    max-height: none;
    padding: 0;
    border: 0;
    background: transparent;
  }

  .chat-server__pane {
    min-height: 60vh;
    border: 0;
    background: transparent;
  }

  .chat-server--show-messages .chat-server__sidebar {
    display: none;
  }

  .chat-server:not(.chat-server--show-messages) .chat-server__pane {
    display: none;
  }
}
</style>
