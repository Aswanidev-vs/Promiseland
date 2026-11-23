<script setup lang="ts">
import { useSlots } from 'vue'
import AppSidebar from './AppSidebar.vue'
import TopBar from './TopBar.vue'
import MobileNav from './MobileNav.vue'
import ToastHost from '@/components/ui/ToastHost.vue'

withDefaults(defineProps<{ wide?: boolean }>(), { wide: false })

const slots = useSlots()
</script>

<template>
  <div class="shell" :class="{ 'shell--wide': wide, 'shell--with-rail': !!slots.rail }">
    <a class="skip-link" href="#main-content">Skip to content</a>

    <aside class="shell__sidebar" aria-label="Primary navigation">
      <slot name="sidebar">
        <AppSidebar />
      </slot>
    </aside>

    <div class="shell__body">
      <header class="shell__topbar">
        <slot name="topbar">
          <TopBar />
        </slot>
      </header>
      <main id="main-content" class="shell__main" :class="{ 'shell__main--wide': wide }">
        <slot />
      </main>
    </div>

    <aside v-if="slots.rail" class="shell__rail" aria-label="Secondary information">
      <slot name="rail" />
    </aside>

    <MobileNav class="shell__mobile-nav" />
    <ToastHost />
  </div>
</template>

<style scoped>
.shell {
  display: grid;
  grid-template-columns: var(--sidebar-w) minmax(0, 1fr);
  gap: var(--layout-gap);
  width: 100%;
  max-width: var(--canvas-max);
  min-height: 100vh;
  margin-inline: auto;
  padding-inline: var(--space-6);
}

.shell--with-rail {
  grid-template-columns: var(--sidebar-w) minmax(0, 1fr) var(--rail-w);
}

.shell__sidebar {
  position: sticky;
  top: 0;
  align-self: start;
  height: 100vh;
  padding-block: var(--space-6);
}

.shell__body {
  min-width: 0;
  padding-bottom: calc(var(--space-16) + var(--bottomnav-h));
}

.shell__topbar {
  position: sticky;
  top: 0;
  z-index: 20;
  margin-inline: calc(var(--space-2) * -1);
  background: var(--ink-900);
}

.shell__main {
  width: 100%;
  max-width: var(--main-max);
  margin-inline: auto;
}

.shell__main--wide {
  max-width: none;
}

.shell__rail {
  position: sticky;
  top: 0;
  align-self: start;
  height: 100vh;
  overflow-y: auto;
  padding-block: var(--space-6);
}

.shell__mobile-nav {
  display: none;
}

.skip-link {
  position: fixed;
  top: var(--space-2);
  left: var(--space-2);
  z-index: 100;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  background: var(--ember);
  color: var(--ink-900);
  transform: translateY(-200%);
}

.skip-link:focus {
  transform: translateY(0);
}

@media (max-width: 1279px) {
  .shell,
  .shell--with-rail {
    grid-template-columns: var(--sidebar-w) minmax(0, 1fr);
  }

  .shell__rail {
    display: none;
  }
}

@media (min-width: 768px) and (max-width: 1023px) {
  .shell,
  .shell--with-rail {
    grid-template-columns: 72px minmax(0, 1fr);
    gap: var(--space-4);
  }
}

@media (max-width: 767px) {
  .shell,
  .shell--with-rail {
    display: block;
    padding-inline: var(--space-4);
  }

  .shell__sidebar {
    display: none;
  }

  .shell__topbar {
    margin-inline: calc(var(--space-4) * -1);
    padding-inline: var(--space-4);
  }

  .shell__body {
    padding-bottom: calc(var(--bottomnav-h) + var(--space-8));
  }

  .shell__mobile-nav {
    display: block;
  }
}
</style>
