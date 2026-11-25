<script setup lang="ts">
import { useSlots } from 'vue'
import AppSidebar from './AppSidebar.vue'
import TopBar from './TopBar.vue'
import MobileNav from './MobileNav.vue'
import GuestBar from './GuestBar.vue'
import ToastHost from '@/components/ui/ToastHost.vue'
import { useGuestStore } from '@/stores/guest'

withDefaults(defineProps<{ wide?: boolean; bare?: boolean }>(), { wide: false, bare: false })

const slots = useSlots()
const guest = useGuestStore()
</script>

<template>
  <div
    class="shell"
    :class="{ 'shell--wide': wide, 'shell--bare': bare, 'shell--with-rail': !!slots.rail }"
  >
    <a class="skip-link" href="#main-content">Skip to content</a>

    <aside v-if="!bare" class="shell__sidebar" aria-label="Primary navigation">
      <slot name="sidebar">
        <AppSidebar />
      </slot>
    </aside>

    <div class="shell__body">
      <header v-if="!bare" class="shell__topbar">
        <slot name="topbar">
          <TopBar />
        </slot>
      </header>
      <GuestBar v-if="!bare && guest.isGuest" />
      <main id="main-content" class="shell__main" :class="{ 'shell__main--wide': wide }">
        <slot />
      </main>
    </div>

    <aside v-if="slots.rail" class="shell__rail" aria-label="Secondary information">
      <slot name="rail" />
    </aside>

    <MobileNav v-if="!bare" class="shell__mobile-nav" />
    <ToastHost />
  </div>
</template>

<style scoped>
/* Canvas: 1280 max, centered. Sidebar 240 flush to the canvas edge with its
   own ink-850 surface + right hairline; main column max 720; rail 280. */
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

/* Bare: public pages that carry their own masthead and footer. */
.shell--bare {
  display: block;
}

.shell--bare .shell__body {
  padding-bottom: 0;
}

.shell__sidebar {
  position: sticky;
  top: 0;
  align-self: start;
  height: 100vh;
  overflow-y: auto;
  /* Bleed into the shell padding so the panel reads as a flush column. */
  margin-inline-start: calc(var(--space-6) * -1);
  padding: var(--space-6) var(--space-3) var(--space-6) var(--space-6);
  border-right: 1px solid var(--border-hairline);
  background: var(--ink-850);
}

.shell__body {
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding-bottom: var(--space-16);
}

.shell__topbar {
  position: sticky;
  top: 0;
  z-index: 20;
  margin-inline: calc(var(--space-6) * -1);
  padding-inline: var(--space-6);
  background: var(--ink-900);
  border-bottom: 1px solid var(--border-hairline);
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
  border-radius: var(--radius-md);
  background: var(--ember);
  color: var(--ink-900);
  text-decoration: none;
  transform: translateY(-200%);
}

.skip-link:focus {
  transform: translateY(0);
}

/* Rail hides below 1280. */
@media (max-width: 1279px) {
  .shell,
  .shell--with-rail {
    grid-template-columns: var(--sidebar-w) minmax(0, 1fr);
  }

  .shell__rail {
    display: none;
  }
}

/* Tablet: sidebar collapses to a 64px icon rail. */
@media (min-width: 768px) and (max-width: 1023px) {
  .shell,
  .shell--with-rail {
    grid-template-columns: 64px minmax(0, 1fr);
    gap: var(--space-4);
  }

  .shell__sidebar {
    padding-right: 0;
  }
}

/* Mobile: single column, fixed bottom tab bar owns the bottom inset. */
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
    padding-bottom: calc(var(--bottomnav-h) + var(--space-8) + env(safe-area-inset-bottom, 0px));
  }

  .shell__mobile-nav {
    display: block;
  }
}
</style>
