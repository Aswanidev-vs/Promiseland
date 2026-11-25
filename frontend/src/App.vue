<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import AppShell from '@/components/layout/AppShell.vue'
import DiscoverRail from '@/components/features/DiscoverRail.vue'
import ErrorView from '@/views/ErrorView.vue'
import { useAuthStore } from '@/stores/auth'
import { useFailureStore } from '@/stores/failure'
import { useMediaQuery } from '@/composables/useBreakpoint'

const route = useRoute()
const auth = useAuthStore()
const failure = useFailureStore()

// Public pages (landing, auth, legal, errors) carry their own masthead.
const isBare = computed(() => route.name === 'landing' || route.meta.bare === true)

// A render crash outranks an unreachable API; both replace the routed view.
// The 503 takeover is limited to app screens: the landing and legal pages do
// not need the API to render, so they stay up and show inline errors instead.
// The rail only exists where the shell shows it, so the requests can wait.
const railVisible = useMediaQuery('(min-width: 1366px)')
const RAIL_SECTIONS: Record<string, Array<'events' | 'creators' | 'communities'>> = {
  home: ['events', 'creators'],
  communities: ['communities', 'events'],
  events: ['creators', 'communities'],
  discover: ['events', 'creators'],
}
const railSections = computed(() => RAIL_SECTIONS[String(route.name)] ?? [])
const showRail = computed(() => !isBare.value && railVisible.value && railSections.value.length > 0)

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
    <template v-if="showRail" #rail>
      <DiscoverRail :sections="railSections" />
    </template>

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
