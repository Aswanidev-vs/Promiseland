<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import AppShell from '@/components/layout/AppShell.vue'
import ErrorView from '@/views/ErrorView.vue'
import { useAuthStore } from '@/stores/auth'
import { useFailureStore } from '@/stores/failure'

const route = useRoute()
const auth = useAuthStore()
const failure = useFailureStore()

// Public pages (landing, auth, legal, errors) carry their own masthead.
const isBare = computed(() => route.name === 'landing' || route.meta.bare === true)

// A render crash outranks an unreachable API; both replace the routed view.
// The 503 takeover is limited to app screens: the landing and legal pages do
// not need the API to render, so they stay up and show inline errors instead.
const failureCode = computed(() => {
  if (failure.renderFailed) return 500
  if (!isBare.value && failure.serviceUnavailable) return 503
  return 0
})

// One session lookup per load: the route guard may already have made it.
onMounted(() => {
  void auth.ensureLoaded()
})
</script>

<template>
  <AppShell :wide="isBare" :bare="isBare">
    <RouterView v-slot="{ Component, route: currentRoute }">
      <Transition name="page" mode="out-in" appear>
        <div :key="currentRoute.path" class="page">
          <component :is="ErrorView" v-if="failureCode" :code="failureCode" />
          <component :is="Component" v-else />
        </div>
      </Transition>
    </RouterView>
  </AppShell>
</template>

<style scoped>
.page {
  min-width: 0;
}
</style>
