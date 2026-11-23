<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import AppShell from '@/components/layout/AppShell.vue'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const auth = useAuthStore()
// The landing page uses the whole canvas; app views stay in the 720px column.
const isLanding = computed(() => route.name === 'landing')

// One session lookup for the whole app; the router guard reuses its result.
onMounted(() => {
  void auth.fetchMe()
})
</script>

<template>
  <AppShell :wide="isLanding">
    <RouterView v-slot="{ Component, route: currentRoute }">
      <Transition name="page" mode="out-in" appear>
        <div :key="currentRoute.path" class="page">
          <component :is="Component" />
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
