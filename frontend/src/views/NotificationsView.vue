<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import { listNotifications, markAllRead, markRead } from '@/api/notifications'
import { useUiStore } from '@/stores/ui'
import type { Notification } from '@/types'
import { errorMessage, formatDay, formatWhen, isForbidden } from '@/components/features/format'

const router = useRouter()
const ui = useUiStore()
const notifications = ref<Notification[]>([])
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const forbidden = ref(false)

const hasUnread = computed(() => notifications.value.some((item) => !item.read_at))

const groups = computed(() => {
  const result: { day: string; items: Notification[] }[] = []
  for (const item of notifications.value) {
    const day = formatDay(item.created_at)
    let group = result.find((entry) => entry.day === day)
    if (!group) {
      group = { day, items: [] }
      result.push(group)
    }
    group.items.push(item)
  }
  return result
})

function verb(type: string): string {
  const verbs: Record<string, string> = {
    follow: 'started following you',
    reply: 'replied to your comment',
    comment: 'commented on your post',
    mention: 'mentioned you',
    reaction: 'reacted to your post',
    bookmark: 'saved your post',
  }
  return verbs[type] || 'sent you an update'
}

function targetOf(item: Notification): string {
  const data = (item.data || {}) as Record<string, unknown>
  if (item.target_type === 'post' || item.target_type === 'comment') {
    const postId = typeof data.post_id === 'number' ? data.post_id : item.target_id
    return `/posts/${postId}`
  }
  if (item.target_type === 'user' || item.target_type === 'follow') {
    const username = typeof data.username === 'string' ? data.username : item.actor?.username
    return username ? `/u/${username}` : '/home'
  }
  if (item.target_type === 'community') {
    const slug = typeof data.slug === 'string' ? data.slug : ''
    return slug ? `/communities/${slug}` : '/communities'
  }
  return '/home'
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  forbidden.value = false
  try {
    const page = await listNotifications({ limit: 50 })
    notifications.value = page.items
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not load notifications. Try again.')
  } finally {
    loading.value = false
  }
}

async function read(item: Notification): Promise<void> {
  if (item.read_at) return
  try {
    await markRead(item.id)
    item.read_at = new Date().toISOString()
  } catch {
    ui.pushToast({ title: 'Notification not marked', message: 'Check your connection and try again.', tone: 'error' })
  }
}

async function readAll(): Promise<void> {
  if (busy.value || !hasUnread.value) return
  busy.value = true
  try {
    await markAllRead()
    const now = new Date().toISOString()
    notifications.value.forEach((item) => (item.read_at = item.read_at || now))
    ui.pushToast({ title: 'All notifications marked as read', tone: 'success' })
  } catch (cause) {
    error.value = errorMessage(cause, 'We could not update your notifications. Try again.')
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="notifications">
    <header class="notifications__head">
      <h1>Notifications</h1>
      <BaseButton variant="ghost" :loading="busy" :disabled="!hasUnread" @click="readAll">
        Mark all as read
      </BaseButton>
    </header>

    <SkeletonList v-if="loading" :rows="5" avatar />

    <EmptyState
      v-else-if="forbidden"
      title="Notifications are not available"
      description="You may need to sign in again to see your activity."
      action-label="Go to home"
      @action="router.push('/home')"
    />

    <EmptyState
      v-else-if="error"
      title="We could not load notifications"
      :description="error"
      action-label="Try again"
      @action="load"
    />

    <EmptyState
      v-else-if="!notifications.length"
      title="Nothing new yet"
      description="Replies, mentions, and new followers will appear here."
      action-label="Explore communities"
      @action="router.push('/communities')"
    />

    <section v-for="group in groups" :key="group.day" class="notifications__group" :aria-label="group.day">
      <h2 class="notifications__day">{{ group.day }}</h2>
      <ul class="notifications__list">
        <li v-for="item in group.items" :key="item.id">
          <RouterLink
            class="notification"
            :class="{ 'notification--unread': !item.read_at }"
            :to="targetOf(item)"
            @click="read(item)"
          >
            <Avatar
              :name="item.actor?.display_name || item.actor?.username || 'PromisedLand'"
              :src="item.actor?.avatar_url"
              size="sm"
            />
            <span class="notification__copy">
              <span class="notification__text">
                <strong>{{ item.actor?.display_name || item.actor?.username || 'PromisedLand' }}</strong>
                {{ verb(item.type) }}
              </span>
              <time v-if="item.created_at" class="notification__time" :datetime="item.created_at">
                {{ formatWhen(item.created_at) }}
              </time>
            </span>
            <span v-if="!item.read_at" class="notification__dot" aria-label="Unread" />
          </RouterLink>
        </li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
.notifications {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.notifications__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.notifications__head h1 {
  font-family: var(--font-display);
  font-weight: 560;
  font-size: var(--text-3xl);
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.notifications__group {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.notifications__day {
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: var(--text-sm);
}

.notifications__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.notification {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-4);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-800);
  color: inherit;
  text-decoration: none;
  transition:
    background-color var(--dur-fast) var(--ease),
    border-color var(--dur-fast) var(--ease);
}

.notification--unread {
  background: var(--ink-850);
}

.notification:hover {
  border-color: var(--border-strong);
}

.notification__copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--space-1);
  min-width: 0;
}

.notification__text {
  color: var(--text-secondary);
}

.notification__text strong {
  color: var(--text-primary);
}

.notification__time {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.notification__dot {
  width: var(--space-2);
  height: var(--space-2);
  border-radius: var(--radius-pill);
  background: var(--ember);
}
</style>
