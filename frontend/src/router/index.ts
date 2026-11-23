import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'landing',
    component: () => import('@/views/LandingView.vue'),
    meta: { title: 'Your fandom. Your people. Your place.' },
  },

  // Public account recovery / verification.
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { title: 'Sign in', guestOnly: true },
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('@/views/RegisterView.vue'),
    meta: { title: 'Create account', guestOnly: true },
  },
  {
    path: '/verify-email',
    name: 'verify-email',
    component: () => import('@/views/VerifyEmailView.vue'),
    meta: { title: 'Verify your email' },
  },
  {
    path: '/reset-password',
    name: 'reset-password',
    component: () => import('@/views/ResetPasswordView.vue'),
    meta: { title: 'Reset password' },
  },

  // Authenticated application.
  {
    path: '/home',
    name: 'home',
    component: () => import('@/views/HomeView.vue'),
    meta: { title: 'Home', requiresAuth: true },
  },
  {
    path: '/onboarding',
    name: 'onboarding',
    component: () => import('@/views/OnboardingView.vue'),
    meta: { title: 'Set up your profile', requiresAuth: true },
  },
  {
    path: '/communities',
    name: 'communities',
    component: () => import('@/views/CommunitiesView.vue'),
    meta: { title: 'Communities', requiresAuth: true },
  },
  {
    path: '/communities/:slug',
    name: 'community',
    component: () => import('@/views/CommunityView.vue'),
    meta: { title: 'Community', requiresAuth: true },
  },
  {
    path: '/posts/new',
    name: 'create-post',
    component: () => import('@/views/CreatePostView.vue'),
    meta: { title: 'Create post', requiresAuth: true },
  },
  {
    path: '/posts/:id',
    name: 'post',
    component: () => import('@/views/PostView.vue'),
    meta: { title: 'Discussion', requiresAuth: true },
  },
  {
    path: '/u/:username',
    name: 'profile',
    component: () => import('@/views/ProfileView.vue'),
    meta: { title: 'Profile', requiresAuth: true },
  },
  {
    path: '/search',
    name: 'search',
    component: () => import('@/views/SearchView.vue'),
    meta: { title: 'Search', requiresAuth: true },
  },
  {
    path: '/notifications',
    name: 'notifications',
    component: () => import('@/views/NotificationsView.vue'),
    meta: { title: 'Notifications', requiresAuth: true },
  },
  {
    path: '/settings',
    name: 'settings',
    component: () => import('@/views/SettingsView.vue'),
    meta: { title: 'Settings', requiresAuth: true },
  },
  {
    path: '/moderation',
    name: 'moderation',
    component: () => import('@/views/ModerationView.vue'),
    meta: { title: 'Moderation', requiresAuth: true },
  },

  // Release 2 — events. Browsing is public; creating and RSVPing need a session.
  {
    path: '/events',
    name: 'events',
    component: () => import('@/views/EventsView.vue'),
    meta: { title: 'Events' },
  },
  {
    path: '/events/new',
    name: 'create-event',
    component: () => import('@/views/CreateEventView.vue'),
    meta: { title: 'Create event', requiresAuth: true },
  },
  {
    path: '/events/:id',
    name: 'event',
    component: () => import('@/views/EventView.vue'),
    // Public view; the RSVP action itself checks auth inside the view.
    meta: { title: 'Event', requiresAuth: false },
  },

  // Release 2 — creators (public directory).
  {
    path: '/creators',
    name: 'creators',
    component: () => import('@/views/CreatorsView.vue'),
    meta: { title: 'Creators' },
  },

  // Release 2 — team-ups (signed-in only).
  {
    path: '/team-ups',
    name: 'team-ups',
    component: () => import('@/views/TeamUpsView.vue'),
    meta: { title: 'Team ups', requiresAuth: true },
  },
  {
    path: '/team-ups/new',
    name: 'create-team-up',
    component: () => import('@/views/CreateTeamUpView.vue'),
    meta: { title: 'Create team up', requiresAuth: true },
  },
  {
    path: '/team-ups/:id',
    name: 'team-up',
    component: () => import('@/views/TeamUpView.vue'),
    meta: { title: 'Team up', requiresAuth: true },
  },

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
  if (typeof document !== 'undefined') {
    document.title = to.meta.title ? `${to.meta.title} — PromisedLand` : 'PromisedLand'
  }

  const auth = useAuthStore()
  if (to.meta.requiresAuth || to.meta.guestOnly) await auth.ensureLoaded()

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  if (to.meta.guestOnly && auth.isAuthenticated) {
    return { name: 'home' }
  }

  return true
})
