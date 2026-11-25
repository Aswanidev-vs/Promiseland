<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import ChatServerList from '@/components/features/ChatServerList.vue'
import ConversationList from '@/components/features/ConversationList.vue'
import CreateServerDialog from '@/components/features/CreateServerDialog.vue'
import NewConversationDialog from '@/components/features/NewConversationDialog.vue'
import ChatIcon from '@/components/features/ChatIcon.vue'
import { listConversations, listServers } from '@/api/chat'
import { useAuthStore } from '@/stores/auth'
import { useBreakpoint } from '@/composables/useBreakpoint'
import { errorMessage, isForbidden } from '@/components/features/format'
import type { ChatServer, Conversation } from '@/types'

const router = useRouter()
const auth = useAuthStore()
const { isMobile } = useBreakpoint()

const servers = ref<ChatServer[]>([])
const conversations = ref<Conversation[]>([])
const serversLoading = ref(true)
const conversationsLoading = ref(true)
const serversError = ref('')
const conversationsError = ref('')
const serversForbidden = ref(false)
const conversationsForbidden = ref(false)
const createOpen = ref(false)
const newOpen = ref(false)
const mobileTab = ref<'servers' | 'conversations'>('servers')

async function loadServers(): Promise<void> {
  serversLoading.value = true
  serversError.value = ''
  serversForbidden.value = false
  try {
    servers.value = await listServers()
  } catch (cause) {
    servers.value = []
    if (isForbidden(cause)) serversForbidden.value = true
    else serversError.value = errorMessage(cause, 'We could not load your servers. Try again.')
  } finally {
    serversLoading.value = false
  }
}

async function loadConversations(): Promise<void> {
  conversationsLoading.value = true
  conversationsError.value = ''
  conversationsForbidden.value = false
  try {
    conversations.value = await listConversations()
  } catch (cause) {
    conversations.value = []
    if (isForbidden(cause)) conversationsForbidden.value = true
    else conversationsError.value = errorMessage(cause, 'We could not load your conversations. Try again.')
  } finally {
    conversationsLoading.value = false
  }
}

function openServer(id: number): void {
  void router.push(`/chat/servers/${id}`)
}

function openConversation(id: number): void {
  void router.push(`/chat/conversations/${id}`)
}

function onServerCreated(server: ChatServer): void {
  servers.value = [server, ...servers.value.filter((item) => item.id !== server.id)]
  openServer(server.id)
}

function onConversationCreated(conversation: Conversation): void {
  conversations.value = [conversation, ...conversations.value.filter((item) => item.id !== conversation.id)]
  openConversation(conversation.id)
}

onMounted(() => {
  void Promise.all([loadServers(), loadConversations()])
})
</script>

<template>
  <div class="chat-home">
    <header class="chat-home__head">
      <div class="chat-home__heading">
        <h1 class="chat-home__title">Chat</h1>
        <p class="chat-home__sub">Servers and conversations in one place.</p>
      </div>
      <RouterLink class="chat-home__back" to="/home">
        <ChatIcon name="back" :size="16" />
        Back to home
      </RouterLink>
    </header>

    <div v-if="isMobile" class="chat-home__body">
      <div class="chat-home__tabs" role="tablist" aria-label="Chat sections">
        <button
          id="chat-tab-servers"
          type="button"
          role="tab"
          :aria-selected="mobileTab === 'servers'"
          aria-controls="chat-panel-servers"
          :class="{ 'chat-home__tab--active': mobileTab === 'servers' }"
          @click="mobileTab = 'servers'"
        >
          Servers
        </button>
        <button
          id="chat-tab-conversations"
          type="button"
          role="tab"
          :aria-selected="mobileTab === 'conversations'"
          aria-controls="chat-panel-conversations"
          :class="{ 'chat-home__tab--active': mobileTab === 'conversations' }"
          @click="mobileTab = 'conversations'"
        >
          Conversations
        </button>
      </div>

      <div
        v-if="mobileTab === 'servers'"
        id="chat-panel-servers"
        role="tabpanel"
        aria-labelledby="chat-tab-servers"
      >
        <ChatServerList
          :servers="servers"
          :loading="serversLoading"
          :error="serversError"
          :forbidden="serversForbidden"
          @select="openServer"
          @create="createOpen = true"
          @retry="loadServers"
        />
      </div>

      <div
        v-else
        id="chat-panel-conversations"
        role="tabpanel"
        aria-labelledby="chat-tab-conversations"
      >
        <ConversationList
          :conversations="conversations"
          :current-user-id="auth.user?.id"
          :loading="conversationsLoading"
          :error="conversationsError"
          :forbidden="conversationsForbidden"
          @select="openConversation"
          @new="newOpen = true"
          @retry="loadConversations"
        />
      </div>
    </div>

    <div v-else class="chat-home__grid">
      <aside class="chat-home__lists" aria-label="Your servers and conversations">
        <ChatServerList
          :servers="servers"
          :loading="serversLoading"
          :error="serversError"
          :forbidden="serversForbidden"
          @select="openServer"
          @create="createOpen = true"
          @retry="loadServers"
        />

        <ConversationList
          :conversations="conversations"
          :current-user-id="auth.user?.id"
          :loading="conversationsLoading"
          :error="conversationsError"
          :forbidden="conversationsForbidden"
          @select="openConversation"
          @new="newOpen = true"
          @retry="loadConversations"
        />
      </aside>

      <section class="chat-home__intro" aria-labelledby="chat-intro-title">
        <img class="chat-home__mark" src="/logo.png" alt="" width="84" height="84" />
        <h2 id="chat-intro-title" class="chat-home__intro-title">
          Pick a server or start a conversation
        </h2>
        <p class="chat-home__intro-copy">
          Channels keep communities organised. Direct messages keep things personal.
        </p>
        <div class="chat-home__intro-actions">
          <BaseButton variant="primary" @click="createOpen = true">Create a server</BaseButton>
          <BaseButton variant="secondary" @click="newOpen = true">Start a conversation</BaseButton>
        </div>
      </section>
    </div>

    <CreateServerDialog v-if="createOpen" @close="createOpen = false" @created="onServerCreated" />
    <NewConversationDialog v-if="newOpen" @close="newOpen = false" @created="onConversationCreated" />
  </div>
</template>

<style scoped>
.chat-home {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  min-width: 0;
}

.chat-home__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-4);
}

.chat-home__title {
  font-size: var(--text-3xl);
}

.chat-home__sub {
  margin-top: var(--space-2);
  color: var(--text-secondary);
}

.chat-home__back {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  min-height: 44px;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  text-decoration: none;
}

.chat-home__body {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
}

.chat-home__tabs {
  display: flex;
  gap: var(--space-2);
  border-bottom: 1px solid var(--border-hairline);
}

.chat-home__tabs button {
  min-height: 44px;
  flex: 1;
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: var(--text-secondary);
  font-weight: 500;
  cursor: pointer;
}

.chat-home__tabs button.chat-home__tab--active {
  border-bottom-color: var(--ember);
  color: var(--text-primary);
}

.chat-home__grid {
  display: grid;
  grid-template-columns: minmax(0, 288px) minmax(0, 1fr);
  align-items: stretch;
  gap: var(--space-4);
  min-width: 0;
}

.chat-home__lists {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
  padding: var(--space-4) var(--space-3);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-850);
}

.chat-home__lists > * + * {
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-hairline);
}

.chat-home__intro {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-4);
  min-width: 0;
  padding: var(--space-10) var(--space-6);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-850);
  text-align: center;
}

.chat-home__mark {
  width: 84px;
  height: 84px;
  border-radius: var(--radius-md);
  object-fit: cover;
}

.chat-home__intro-title {
  max-width: 22ch;
  font-size: var(--text-xl);
  font-weight: 600;
  letter-spacing: -0.01em;
  line-height: 1.25;
}

.chat-home__intro-copy {
  max-width: 42ch;
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.chat-home__intro-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: var(--space-3);
  padding-top: var(--space-2);
}

@media (max-width: 1023px) {
  .chat-home__grid {
    grid-template-columns: minmax(0, 240px) minmax(0, 1fr);
  }
}

</style>
