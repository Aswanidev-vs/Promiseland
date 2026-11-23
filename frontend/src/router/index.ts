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
