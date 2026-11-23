<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AppShell from '@/components/layout/AppShell.vue'

const route = useRoute()
// The landing page uses the whole canvas; app views stay in the 720px column.
const isLanding = computed(() => route.name === 'landing')
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
