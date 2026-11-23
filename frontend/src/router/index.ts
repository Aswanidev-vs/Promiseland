import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'landing',
    component: () => import('@/views/LandingView.vue'),
    meta: { title: 'Your fandom. Your people. Your place.' },
  },

  // ─────────────────────────────────────────────────────────────────────────────
  // R1 agents: register the real application routes here and remove the
  // placeholder disabled nav items in AppSidebar.vue / MobileNav.vue:
  //   /home (HomeFeedView), /communities, /c/:slug, /events,
  //   /discover, /notifications, /u/:username, /search,
  //   /login (meta.guestOnly), /register (meta.guestOnly),
  //   /settings, /moderation — and mark account-only views with
  //   meta: { requiresAuth: true }.
  // R2 agents: add /events/:id, /events/new, /team-ups.
  // R3 agents: add /chat, /chat/servers/:id and the WebSocket client.
  // ─────────────────────────────────────────────────────────────────────────────

  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFoundView.vue'),
    meta: { title: 'Page not found' },
  },
]

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach(async (to) => {
  document.title = to.meta.title ? `${to.meta.title} — PromisedLand` : 'PromisedLand'

  const auth = useAuthStore()
  if (to.meta.requiresAuth || to.meta.guestOnly) await auth.ensureLoaded()

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    // R1: replace with { name: 'login', query: { next: to.fullPath } }.
    return { name: 'landing', query: { next: to.fullPath } }
  }

  if (to.meta.guestOnly && auth.isAuthenticated) {
    // R1: replace with { name: 'home' }.
    return { name: 'landing' }
  }

  return true
})
