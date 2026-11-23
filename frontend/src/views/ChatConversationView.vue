<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import ChatComposer from '@/components/features/ChatComposer.vue'
import ChatIcon from '@/components/features/ChatIcon.vue'
import ChatMessageList from '@/components/features/ChatMessageList.vue'
import PresenceDot from '@/components/features/PresenceDot.vue'
import ReportDialog from '@/components/features/ReportDialog.vue'
import TypingIndicator from '@/components/features/TypingIndicator.vue'
import {
  deleteMessage,
  getPresence,
  listConversations,
  listMessages,
  reactToMessage,
  reportMessage,
  sendMessage,
} from '@/api/chat'
import { useChatSocket } from '@/composables/useChatSocket'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { errorMessage, isForbidden, isNotFound, plural } from '@/components/features/format'
import {
  conversationLabel,
  conversationMembers,
  mergeMessages,
  normalizePresence,
  normalizeSocketMessages,
  normalizeTyping,
} from '@/components/features/chatMessageUtils'
import type { ChatMessage, Conversation } from '@/types'

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

const conversationId = computed(() => Number(route.params.id))
const scope = computed(() => ({ conversationId: conversationId.value }))

const conversation = ref<Conversation | null>(null)
const messages = ref<ChatMessage[]>([])
const presence = ref<Record<number, string>>({})

const loadingConversation = ref(true)
const loadingMessages = ref(false)
const loadingMore = ref(false)
const conversationError = ref('')
const messagesError = ref('')
const forbidden = ref(false)
const notFound = ref(false)
const cursor = ref('')
const replyTo = ref<ChatMessage | null>(null)
const reportTarget = ref<ChatMessage | null>(null)
const actionBusy = ref(false)
const online = ref(typeof navigator === 'undefined' ? true : navigator.onLine)
const composer = ref<InstanceType<typeof ChatComposer> | null>(null)

const members = computed(() => (conversation.value ? conversationMembers(conversation.value, auth.user?.id) : []))
const authors = computed(() => {
  const people = messages.value.map((message) => message.author).filter(Boolean) as NonNullable<ChatMessage['author']>[]
  for (const member of members.value) if (!people.some((person) => person.id === member.id)) people.push(member)
  return people
})
const label = computed(() =>
  conversation.value ? conversationLabel(conversation.value, auth.user?.id) : 'Conversation',
)
const typingNames = computed(() => normalizeTyping(socketTyping.value, scope.value, authors.value))

function onOnline(): void {
  online.value = true
}

function onOffline(): void {
  online.value = false
}

async function loadConversation(): Promise<void> {
  loadingConversation.value = true
  conversationError.value = ''
  forbidden.value = false
  notFound.value = false
  try {
    const list = await listConversations()
    const found = list.find((item) => item.id === conversationId.value) || null
    conversation.value = found
    if (!found) notFound.value = true
    else void loadPresence()
  } catch (cause) {
    conversation.value = null
    if (isForbidden(cause)) forbidden.value = true
    else if (isNotFound(cause)) notFound.value = true
    else conversationError.value = errorMessage(cause, 'We could not load this conversation. Try again.')
  } finally {
    loadingConversation.value = false
  }
}

async function loadPresence(): Promise<void> {
  const ids = [...new Set([...members.value.map((member) => member.id), ...authors.value.map((author) => author.id)])]
  if (!ids.length) return
  try {
    const initial = await getPresence(ids)
    presence.value = { ...initial, ...presence.value }
  } catch {
    // Presence is optional and must not block reading messages.
  }
}

async function loadMessages(): Promise<void> {
  const id = conversationId.value
  loadingMessages.value = true
  messagesError.value = ''
  messages.value = []
  cursor.value = ''
  try {
    const page = await listMessages({ conversationId: id, limit: 50 })
    messages.value = mergeMessages([], page.items || [])
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
  if (!cursor.value || loadingMore.value) return
  loadingMore.value = true
  try {
    const page = await listMessages({ conversationId: conversationId.value, limit: 50, cursor: cursor.value })
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

async function send(body: string): Promise<void> {
  try {
    const message = await sendMessage({ conversationId: conversationId.value }, body, replyTo.value?.id)
    messages.value = mergeMessages(messages.value, [message])
    replyTo.value = null
  } catch (cause) {
    ui.pushToast({ title: 'Message not sent', message: errorMessage(cause), tone: 'error' })
  }
}

function onTyping(isTyping: boolean): void {
  void sendTyping({ conversationId: conversationId.value }, isTyping)
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

watch(conversationId, () => {
  subscribe(scope.value)
  void Promise.all([loadConversation(), loadMessages()])
})

onMounted(() => {
  connect()
  subscribe(scope.value)
  void Promise.all([loadConversation(), loadMessages()])
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
  <div class="conversation">
    <header class="conversation__head">
      <RouterLink class="conversation__back" to="/chat" aria-label="Back to chat">
        <ChatIcon name="back" :size="18" />
      </RouterLink>

      <div v-if="conversation" class="conversation__identity">
        <ul class="conversation__avatars" :aria-label="plural(members.length, 'member')">
          <li v-for="member in members.slice(0, 4)" :key="member.id" class="conversation__avatar">
            <Avatar :name="member.display_name || member.username" :src="member.avatar_url" size="sm" />
            <PresenceDot :status="presence[member.id]" />
          </li>
        </ul>
        <div class="conversation__titles">
          <h1 class="conversation__title">{{ label }}</h1>
          <p class="conversation__meta">
            <ChatIcon v-if="conversation.type === 'group'" name="group" :size="14" />
            <span>{{ plural(members.length, 'member') }}</span>
          </p>
        </div>
      </div>
      <h1 v-else class="conversation__title">Conversation</h1>
    </header>

    <p v-if="!online" class="conversation__offline" role="status">
      <ChatIcon name="offline" :size="16" />
      Reconnecting. Messages will catch up when the connection returns.
    </p>

    <SkeletonList v-if="loadingConversation" :rows="3" />

    <EmptyState
      v-else-if="notFound"
      title="This conversation is not available"
      description="It may have been removed, or the link may be wrong."
      action-label="Back to chat"
      @action="router.push('/chat')"
    />

    <EmptyState
      v-else-if="forbidden"
      title="You do not have access to this conversation"
      description="You may need to be added back to the group."
      action-label="Back to chat"
      @action="router.push('/chat')"
    />

    <EmptyState
      v-else-if="conversationError"
      title="We could not load this conversation"
      :description="conversationError"
      action-label="Try again"
      @action="loadConversation"
    />

    <section v-else class="conversation__pane" aria-label="Messages">
      <ChatMessageList
        :messages="messages"
        :current-user-id="auth.user?.id"
        :presence="presence"
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

      <div v-if="cursor" class="conversation__more">
        <BaseButton variant="ghost" size="sm" :loading="loadingMore" @click="loadMore">Load earlier messages</BaseButton>
      </div>

      <div class="conversation__compose">
        <TypingIndicator :names="typingNames" />
        <ChatComposer
          ref="composer"
          :replying-to="replyTo ? (replyTo.author?.display_name || replyTo.author?.username || 'member') : null"
          :placeholder="`Message ${label}`"
          @send="send"
          @typing="onTyping"
          @cancel-reply="replyTo = null"
        />
      </div>
    </section>

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
.conversation {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
}

.conversation__head {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding-bottom: var(--space-4);
  border-bottom: 1px solid var(--border-hairline);
}

.conversation__back {
  display: inline-flex;
  width: 44px;
  height: 44px;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  color: var(--text-secondary);
}

.conversation__identity {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: var(--space-3);
}

.conversation__avatars {
  display: flex;
  flex: none;
  padding-left: var(--space-2);
}

.conversation__avatar {
  position: relative;
  margin-left: calc(var(--space-2) * -1);
  border-radius: var(--radius-pill);
  box-shadow: 0 0 0 2px var(--ink-900);
}

.conversation__avatar :deep(.presence) {
  position: absolute;
  right: -2px;
  bottom: -2px;
  box-shadow: 0 0 0 2px var(--ink-900);
}

.conversation__titles {
  min-width: 0;
}

.conversation__title {
  overflow: hidden;
  font-size: var(--text-2xl);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conversation__meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.conversation__offline {
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

.conversation__pane {
  display: flex;
  flex-direction: column;
  min-height: min(72vh, 640px);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-850);
}

.conversation__more {
  display: flex;
  justify-content: center;
  padding: var(--space-2);
}

.conversation__compose {
  margin-top: auto;
}

@media (max-width: 767px) {
  .conversation__pane {
    min-height: 60vh;
    border: 0;
    background: transparent;
  }

  .conversation__title {
    font-size: var(--text-xl);
  }
}
</style>
