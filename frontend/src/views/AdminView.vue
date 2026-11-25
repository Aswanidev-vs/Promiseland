<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import {
  getAdminOverview,
  listAdminCommunities,
  listAdminEvents,
  listAdminPosts,
  listAdminUsers,
  setAdminUserActive,
  type AdminCommunity,
  type AdminEvent,
  type AdminOverview,
  type AdminPost,
  type AdminUser,
} from '@/api/admin'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { errorMessage } from '@/components/features/format'

type Tab = 'overview' | 'users' | 'posts' | 'communities' | 'events'

interface Table<T> {
  items: T[]
  cursor: string
  loading: boolean
  error: string
  query: string
}

function createTable<T>(): Table<T> {
  return { items: [], cursor: '', loading: false, error: '', query: '' }
}

const auth = useAuthStore()
const ui = useUiStore()
const isStaff = computed(() => auth.user?.is_staff === true)

const tabs: Array<{ value: Tab; label: string }> = [
  { value: 'overview', label: 'Overview' },
  { value: 'users', label: 'Users' },
  { value: 'posts', label: 'Posts' },
  { value: 'communities', label: 'Communities' },
  { value: 'events', label: 'Events' },
]
const tab = ref<Tab>('overview')

const overview = ref<AdminOverview | null>(null)
const overviewError = ref('')
const overviewLoading = ref(false)

const users = reactive(createTable<AdminUser>())
const posts = reactive(createTable<AdminPost>())
const communities = reactive(createTable<AdminCommunity>())
const events = reactive(createTable<AdminEvent>())

const busyUserId = ref(0)

function formatDate(value: string): string {
  const date = new Date(value.includes('T') ? value : `${value.replace(' ', 'T')}Z`)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(undefined, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  }).format(date)
}

function formatBytes(value: number): string {
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${Math.round(value / 1024)} KB`
  return `${(value / (1024 * 1024)).toFixed(1)} MB`
}

async function loadOverview(): Promise<void> {
  overviewLoading.value = true
  overviewError.value = ''
  try {
    overview.value = await getAdminOverview()
  } catch (cause) {
    overviewError.value = errorMessage(cause, 'We could not load the overview. Try again.')
  } finally {
    overviewLoading.value = false
  }
}

async function loadUsers(reset = false): Promise<void> {
  await loadTable(users, listAdminUsers, reset)
}
async function loadPosts(reset = false): Promise<void> {
  await loadTable(posts, listAdminPosts, reset)
}
async function loadCommunities(reset = false): Promise<void> {
  await loadTable(communities, listAdminCommunities, reset)
}
async function loadEvents(reset = false): Promise<void> {
  await loadTable(events, listAdminEvents, reset)
}

async function loadTable<T>(
  table: Table<T>,
  fetcher: (params: { q?: string; limit?: number; cursor?: string }) => Promise<{ items: T[]; next_cursor: string }>,
  reset = false,
): Promise<void> {
  if (table.loading) return
  const more = !reset && table.cursor !== ''
  if (reset) {
    table.items = []
    table.cursor = ''
  }
  table.loading = true
  table.error = ''
  try {
    const page = await fetcher({
      ...(table.query.trim() ? { q: table.query.trim() } : {}),
      limit: 20,
      ...(more ? { cursor: table.cursor } : {}),
    })
    table.items = more ? [...table.items, ...page.items] : page.items
    table.cursor = page.next_cursor || ''
  } catch (cause) {
    table.error = errorMessage(cause, 'We could not load that table. Try again.')
  } finally {
    table.loading = false
  }
}

async function toggleUser(user: AdminUser): Promise<void> {
  busyUserId.value = user.id
  try {
    const updated = await setAdminUserActive(user.id, !user.is_active)
    user.is_active = updated.is_active
    ui.pushToast({
      title: updated.is_active ? `Reinstated @${user.username}` : `Suspended @${user.username}`,
      message: updated.is_active ? 'Their sessions can sign in again.' : 'Their sessions were ended.',
      tone: 'success',
    })
  } catch (cause) {
    ui.pushToast({
      title: 'Could not change that account',
      message: errorMessage(cause, 'Try again in a moment.'),
      tone: 'error',
    })
  } finally {
    busyUserId.value = 0
  }
}

async function openTab(next: Tab): Promise<void> {
  tab.value = next
  if (next === 'overview' && !overview.value) await loadOverview()
  if (next === 'users' && !users.items.length) await loadUsers(true)
  if (next === 'posts' && !posts.items.length) await loadPosts(true)
  if (next === 'communities' && !communities.items.length) await loadCommunities(true)
  if (next === 'events' && !events.items.length) await loadEvents(true)
}

onMounted(async () => {
  if (isStaff.value) await loadOverview()
})
</script>

<template>
  <div class="admin">
    <header class="admin__head">
      <h1 class="admin__title">Admin</h1>
      <p class="admin__sub">Accounts, content and events across Promiseland. Staff only.</p>
    </header>

    <EmptyState
      v-if="!isStaff"
      title="This console is for staff"
      description="Your account does not hold a platform staff grant. Ask an administrator to add you."
      action-label="Go to home"
      @action="$router.push('/home')"
    />

    <template v-else>
      <nav class="admin__tabs" role="tablist" aria-label="Admin sections">
        <button
          v-for="item in tabs"
          :key="item.value"
          type="button"
          role="tab"
          :aria-selected="tab === item.value"
          :class="{ 'admin__tab--active': tab === item.value }"
          @click="openTab(item.value)"
        >
          {{ item.label }}
        </button>
      </nav>

      <!-- Overview -->
      <section v-if="tab === 'overview'" class="admin__panel" aria-label="Overview">
        <SkeletonList v-if="overviewLoading && !overview" :rows="3" />

        <EmptyState
          v-else-if="overviewError"
          title="We could not load the overview"
          :description="overviewError"
          action-label="Try again"
          @action="loadOverview"
        />

        <template v-else-if="overview">
          <dl class="admin__counts">
            <div class="admin__count">
              <dt>Accounts</dt>
              <dd class="mono">{{ overview.counts.users }}</dd>
            </div>
            <div class="admin__count">
              <dt>Active</dt>
              <dd class="mono">{{ overview.counts.active_users }}</dd>
            </div>
            <div class="admin__count">
              <dt>Suspended</dt>
              <dd class="mono">{{ overview.counts.suspended_users }}</dd>
            </div>
            <div class="admin__count">
              <dt>Sessions</dt>
              <dd class="mono">{{ overview.counts.active_sessions }}</dd>
            </div>
            <div class="admin__count">
              <dt>Posts</dt>
              <dd class="mono">{{ overview.counts.posts }}</dd>
            </div>
            <div class="admin__count">
              <dt>Comments</dt>
              <dd class="mono">{{ overview.counts.comments }}</dd>
            </div>
            <div class="admin__count">
              <dt>Communities</dt>
              <dd class="mono">{{ overview.counts.communities }}</dd>
            </div>
            <div class="admin__count">
              <dt>Events</dt>
              <dd class="mono">{{ overview.counts.events }}</dd>
            </div>
            <div class="admin__count">
              <dt>Media files</dt>
              <dd class="mono">{{ overview.counts.media }}</dd>
            </div>
            <div class="admin__count">
              <dt>Media stored</dt>
              <dd class="mono">{{ formatBytes(overview.counts.media_bytes) }}</dd>
            </div>
            <div class="admin__count">
              <dt>Open reports</dt>
              <dd class="mono">{{ overview.counts.open_reports }}</dd>
            </div>
            <div class="admin__count">
              <dt>Pending joins</dt>
              <dd class="mono">{{ overview.counts.pending_memberships }}</dd>
            </div>
          </dl>

          <div class="admin__columns">
            <section class="admin__list" aria-labelledby="admin-new-users">
              <h2 id="admin-new-users" class="admin__heading">New accounts</h2>
              <ul>
                <li v-for="user in overview.recent_users" :key="user.id" class="admin__row">
                  <RouterLink class="admin__link" :to="`/u/${user.username}`">
                    {{ user.display_name }}
                  </RouterLink>
                  <span class="admin__meta mono">@{{ user.username }}</span>
                  <span class="admin__meta">{{ formatDate(user.created_at) }}</span>
                </li>
                <li v-if="!overview.recent_users.length" class="admin__meta">No accounts yet.</li>
              </ul>
            </section>

            <section class="admin__list" aria-labelledby="admin-recent-posts">
              <h2 id="admin-recent-posts" class="admin__heading">Recent posts</h2>
              <ul>
                <li v-for="post in overview.recent_posts" :key="post.id" class="admin__row">
                  <RouterLink class="admin__link" :to="`/posts/${post.id}`">{{ post.title }}</RouterLink>
                  <span class="admin__meta mono">@{{ post.author }}</span>
                  <span class="admin__meta">{{ post.community || 'Profile' }}</span>
                </li>
                <li v-if="!overview.recent_posts.length" class="admin__meta">No posts yet.</li>
              </ul>
            </section>

            <section class="admin__list" aria-labelledby="admin-open-reports">
              <h2 id="admin-open-reports" class="admin__heading">Open reports</h2>
              <ul>
                <li v-for="report in overview.open_reports" :key="report.id" class="admin__row">
                  <span class="admin__link">{{ report.reason }}</span>
                  <span class="admin__meta mono">{{ report.target_type }} #{{ report.target_id }}</span>
                  <span class="admin__meta">{{ formatDate(report.created_at) }}</span>
                </li>
                <li v-if="!overview.open_reports.length" class="admin__meta">Nothing open.</li>
              </ul>
              <RouterLink class="admin__more" to="/moderation">Open the moderation queue</RouterLink>
            </section>
          </div>
        </template>
      </section>

      <!-- Users -->
      <section v-else-if="tab === 'users'" class="admin__panel" aria-label="Users">
        <form class="admin__search" role="search" @submit.prevent="loadUsers(true)">
          <label class="sr-only" for="admin-users-q">Search accounts</label>
          <input
            id="admin-users-q"
            v-model="users.query"
            class="admin__input"
            type="search"
            placeholder="Name, username or email"
          />
          <BaseButton type="submit" variant="secondary" size="sm">Search</BaseButton>
        </form>

        <p v-if="users.error" class="admin__error" role="alert">{{ users.error }}</p>
        <SkeletonList v-else-if="users.loading && !users.items.length" :rows="4" />
        <p v-else-if="!users.items.length" class="admin__empty">No accounts match.</p>

        <div v-else class="admin__scroll">
          <table class="admin__table">
            <thead>
              <tr>
                <th scope="col">Account</th>
                <th scope="col">Email</th>
                <th scope="col" class="admin__num">Posts</th>
                <th scope="col" class="admin__num">Sessions</th>
                <th scope="col">State</th>
                <th scope="col">Joined</th>
                <th scope="col"><span class="sr-only">Actions</span></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="user in users.items" :key="user.id">
                <td>
                  <RouterLink class="admin__link" :to="`/u/${user.username}`">
                    {{ user.display_name }}
                  </RouterLink>
                  <span class="admin__meta mono">@{{ user.username }}</span>
                  <span v-if="user.is_staff" class="admin__badge">Staff</span>
                </td>
                <td class="admin__meta">{{ user.email }}</td>
                <td class="admin__num mono">{{ user.post_count }}</td>
                <td class="admin__num mono">{{ user.active_sessions }}</td>
                <td>
                  <span :class="['admin__state', user.is_active ? 'admin__state--ok' : 'admin__state--alert']">
                    {{ user.is_active ? 'Active' : 'Suspended' }}
                  </span>
                </td>
                <td class="admin__meta">{{ formatDate(user.created_at) }}</td>
                <td class="admin__actions-cell">
                  <BaseButton
                    size="sm"
                    :variant="user.is_active ? 'secondary' : 'primary'"
                    :loading="busyUserId === user.id"
                    :disabled="user.id === auth.user?.id"
                    @click="toggleUser(user)"
                  >
                    {{ user.is_active ? 'Suspend' : 'Reinstate' }}
                  </BaseButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <BaseButton
          v-if="users.cursor"
          class="admin__load"
          variant="secondary"
          size="sm"
          :loading="users.loading"
          @click="loadUsers()"
        >
          Load more
        </BaseButton>
      </section>

      <!-- Posts -->
      <section v-else-if="tab === 'posts'" class="admin__panel" aria-label="Posts">
        <form class="admin__search" role="search" @submit.prevent="loadPosts(true)">
          <label class="sr-only" for="admin-posts-q">Search posts</label>
          <input
            id="admin-posts-q"
            v-model="posts.query"
            class="admin__input"
            type="search"
            placeholder="Title"
          />
          <BaseButton type="submit" variant="secondary" size="sm">Search</BaseButton>
        </form>

        <p v-if="posts.error" class="admin__error" role="alert">{{ posts.error }}</p>
        <SkeletonList v-else-if="posts.loading && !posts.items.length" :rows="4" />
        <p v-else-if="!posts.items.length" class="admin__empty">No posts match.</p>

        <div v-else class="admin__scroll">
          <table class="admin__table">
            <thead>
              <tr>
                <th scope="col">Post</th>
                <th scope="col">Author</th>
                <th scope="col">Community</th>
                <th scope="col" class="admin__num">Comments</th>
                <th scope="col" class="admin__num">Score</th>
                <th scope="col">Posted</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="post in posts.items" :key="post.id">
                <td>
                  <RouterLink class="admin__link" :to="`/posts/${post.id}`">{{ post.title }}</RouterLink>
                  <span class="admin__meta">{{ post.type }}</span>
                </td>
                <td class="admin__meta mono">@{{ post.author }}</td>
                <td class="admin__meta">{{ post.community || 'Profile' }}</td>
                <td class="admin__num mono">{{ post.comment_count }}</td>
                <td class="admin__num mono">{{ post.vote_score }}</td>
                <td class="admin__meta">{{ formatDate(post.created_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <BaseButton
          v-if="posts.cursor"
          class="admin__load"
          variant="secondary"
          size="sm"
          :loading="posts.loading"
          @click="loadPosts()"
        >
          Load more
        </BaseButton>
      </section>

      <!-- Communities -->
      <section v-else-if="tab === 'communities'" class="admin__panel" aria-label="Communities">
        <form class="admin__search" role="search" @submit.prevent="loadCommunities(true)">
          <label class="sr-only" for="admin-communities-q">Search communities</label>
          <input
            id="admin-communities-q"
            v-model="communities.query"
            class="admin__input"
            type="search"
            placeholder="Name or slug"
          />
          <BaseButton type="submit" variant="secondary" size="sm">Search</BaseButton>
        </form>

        <p v-if="communities.error" class="admin__error" role="alert">{{ communities.error }}</p>
        <SkeletonList v-else-if="communities.loading && !communities.items.length" :rows="4" />
        <p v-else-if="!communities.items.length" class="admin__empty">No communities match.</p>

        <div v-else class="admin__scroll">
          <table class="admin__table">
            <thead>
              <tr>
                <th scope="col">Community</th>
                <th scope="col">Owner</th>
                <th scope="col" class="admin__num">Members</th>
                <th scope="col" class="admin__num">Posts</th>
                <th scope="col">Visibility</th>
                <th scope="col">Created</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in communities.items" :key="item.id">
                <td>
                  <RouterLink class="admin__link" :to="`/communities/${item.slug}`">{{ item.name }}</RouterLink>
                  <span class="admin__meta mono">{{ item.slug }}</span>
                </td>
                <td class="admin__meta mono">@{{ item.owner }}</td>
                <td class="admin__num mono">{{ item.member_count }}</td>
                <td class="admin__num mono">{{ item.post_count }}</td>
                <td class="admin__meta">{{ item.visibility }}</td>
                <td class="admin__meta">{{ formatDate(item.created_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <BaseButton
          v-if="communities.cursor"
          class="admin__load"
          variant="secondary"
          size="sm"
          :loading="communities.loading"
          @click="loadCommunities()"
        >
          Load more
        </BaseButton>
      </section>

      <!-- Events -->
      <section v-else class="admin__panel" aria-label="Events">
        <form class="admin__search" role="search" @submit.prevent="loadEvents(true)">
          <label class="sr-only" for="admin-events-q">Search events</label>
          <input
            id="admin-events-q"
            v-model="events.query"
            class="admin__input"
            type="search"
            placeholder="Title or city"
          />
          <BaseButton type="submit" variant="secondary" size="sm">Search</BaseButton>
        </form>

        <p v-if="events.error" class="admin__error" role="alert">{{ events.error }}</p>
        <SkeletonList v-else-if="events.loading && !events.items.length" :rows="4" />
        <p v-else-if="!events.items.length" class="admin__empty">No events match.</p>

        <div v-else class="admin__scroll">
          <table class="admin__table">
            <thead>
              <tr>
                <th scope="col">Event</th>
                <th scope="col">Organizer</th>
                <th scope="col">Where</th>
                <th scope="col">Starts</th>
                <th scope="col" class="admin__num">Going</th>
                <th scope="col">Status</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in events.items" :key="item.id">
                <td>
                  <RouterLink class="admin__link" :to="`/events/${item.id}`">{{ item.title }}</RouterLink>
                  <span class="admin__meta">{{ item.event_type }}</span>
                </td>
                <td class="admin__meta mono">@{{ item.organizer }}</td>
                <td class="admin__meta">{{ item.is_online ? 'Online' : item.city || 'Not set' }}</td>
                <td class="admin__meta">{{ formatDate(item.starts_at) }}</td>
                <td class="admin__num mono">{{ item.going_count }}</td>
                <td class="admin__meta">{{ item.status }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <BaseButton
          v-if="events.cursor"
          class="admin__load"
          variant="secondary"
          size="sm"
          :loading="events.loading"
          @click="loadEvents()"
        >
          Load more
        </BaseButton>
      </section>
    </template>
  </div>
</template>

<style scoped>
.admin {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.admin__head {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.admin__title {
  font-family: var(--font-display);
  font-size: var(--text-3xl);
  font-weight: 560;
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.admin__sub {
  color: var(--text-secondary);
}

.admin__tabs {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
  border-bottom: 1px solid var(--border-hairline);
}

.admin__tabs button {
  min-height: 44px;
  padding: 0 var(--space-3);
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: var(--text-secondary);
  font: inherit;
  font-weight: 500;
  cursor: pointer;
}

.admin__tabs button.admin__tab--active {
  border-bottom-color: var(--ember);
  color: var(--text-primary);
}

.admin__panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.admin__counts {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  margin: 0;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--border-hairline);
  overflow: hidden;
}

.admin__count {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-4);
  background: var(--ink-850);
}

.admin__count dt {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.admin__count dd {
  margin: 0;
  font-size: var(--text-xl);
  font-variant-numeric: tabular-nums;
}

.admin__columns {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-5);
}

.admin__list ul {
  display: flex;
  flex-direction: column;
}

.admin__row {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--space-2) var(--space-3);
  padding-block: var(--space-2);
  border-bottom: 1px solid var(--border-hairline);
}

.admin__row:last-child {
  border-bottom: 0;
}

.admin__heading {
  margin-bottom: var(--space-2);
  font-size: var(--text-sm);
  font-weight: 600;
}

.admin__link {
  overflow: hidden;
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
  text-decoration: none;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.admin__link:hover {
  color: var(--tide);
}

.admin__meta {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.admin__badge {
  margin-left: var(--space-2);
  padding: 1px var(--space-2);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-pill);
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.admin__search {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
}

.admin__input {
  min-height: 44px;
  flex: 1 1 240px;
  min-width: 0;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
}

.admin__input:focus {
  border-color: var(--ember);
  box-shadow: 0 0 0 3px var(--ember-soft);
  outline: none;
}

.admin__scroll {
  overflow-x: auto;
}

.admin__table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}

.admin__table th {
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--border-strong);
  color: var(--text-secondary);
  font-size: var(--text-xs);
  font-weight: 500;
  text-align: left;
  white-space: nowrap;
}

.admin__table td {
  padding: var(--space-3);
  border-bottom: 1px solid var(--border-hairline);
  vertical-align: top;
}

.admin__table td .admin__meta {
  display: block;
}

.admin__num {
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.admin__actions-cell {
  text-align: right;
  white-space: nowrap;
}

.admin__state {
  font-size: var(--text-xs);
  font-weight: 500;
}

.admin__state--ok {
  color: var(--ok);
}

.admin__state--alert {
  color: var(--alert);
}

.admin__empty,
.admin__error {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.admin__error {
  color: var(--alert);
}

.admin__more {
  display: inline-flex;
  align-items: center;
  margin-top: var(--space-2);
  min-height: 44px;
  color: var(--tide);
  font-size: var(--text-sm);
  text-decoration: none;
}

.admin__more:hover {
  color: var(--text-primary);
}

.admin__load {
  align-self: flex-start;
}

@media (max-width: 1023px) {
  .admin__counts {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .admin__columns {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 599px) {
  .admin__counts {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
