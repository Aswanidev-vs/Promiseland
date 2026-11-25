<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import AppIcon from '@/components/ui/AppIcon.vue'
import type { IconName } from '@/components/ui/icons'
import Avatar from '@/components/ui/Avatar.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import TagChip from '@/components/ui/TagChip.vue'
import type { Creator, Post } from '@/types'
import { plural } from './format'
import { creatorCategoryLabel } from './formatR2'

const props = defineProps<{ creator: Creator }>()

const work = computed<Post[]>(() => (props.creator.featured_posts || []).slice(0, 3))
const slots = computed(() => [0, 1, 2].map((index) => work.value[index] || null))
const categories = computed(() => (props.creator.categories || []).slice(0, 3))

function glyphFor(category: string): IconName {
  if (category === 'photography') return 'eye'
  if (category === 'collection') return 'discover'
  return 'creative'
}

function thumbGlyph(post: Post | null, index: number): IconName {
  if (post?.creator_category) return glyphFor(post.creator_category)
  return glyphFor(categories.value[index] || categories.value[0] || 'other')
}
</script>

<template>
  <BaseCard interactive class="creator-card" :padded="false">
    <RouterLink class="creator-card__link" :to="`/u/${creator.user?.username || ''}`">
      <div class="creator-card__identity">
        <Avatar
          :name="creator.user?.display_name || creator.user?.username || 'Creator'"
          :src="creator.user?.avatar_url"
          size="md"
        />
        <div class="creator-card__who">
          <h2 class="creator-card__name">{{ creator.user?.display_name || creator.user?.username || 'Creator' }}</h2>
          <p class="creator-card__handle mono">@{{ creator.user?.username || 'creator' }}</p>
          <div v-if="categories.length" class="creator-card__categories">
            <TagChip v-for="category in categories" :key="category" :label="creatorCategoryLabel(category)" />
          </div>
        </div>
      </div>

      <div class="creator-card__work" aria-label="Featured work">
        <div v-for="(post, index) in slots" :key="index" class="creator-card__thumb">
          <img
            v-if="post?.media?.[0]?.url"
            :src="post.media[0].url"
            :alt="post.title || 'Featured work'"
            loading="lazy"
          />
          <span v-else class="creator-card__glyph" aria-hidden="true">
            <AppIcon :name="thumbGlyph(post, index)" :size="28" />
          </span>
        </div>
      </div>

      <div class="creator-card__foot">
        <span class="creator-card__count mono">{{ plural(creator.post_count || 0, 'post') }}</span>
        <span class="creator-card__count mono">{{ plural(creator.follower_count || 0, 'follower') }}</span>
        <span class="creator-card__view">View portfolio</span>
      </div>
    </RouterLink>
  </BaseCard>
</template>

<style scoped>
.creator-card {
  overflow: hidden;
}

.creator-card__link {
  display: flex;
  height: 100%;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-5);
  color: inherit;
  text-decoration: none;
}

.creator-card__identity {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
}

.creator-card__who {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: var(--space-2);
}

.creator-card__name {
  overflow: hidden;
  color: var(--text-primary);
  font-family: var(--font-display);
  font-size: var(--text-lg);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.creator-card__link:hover .creator-card__name {
  color: var(--tide);
}

.creator-card__handle {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.creator-card__categories {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}

.creator-card__work {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-2);
}

.creator-card__thumb {
  display: flex;
  aspect-ratio: 4 / 3;
  overflow: hidden;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-sm);
  background: var(--ink-850);
}

.creator-card__thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.creator-card__glyph {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--tide);
  opacity: 0.75;
}

.creator-card__foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2) var(--space-4);
  margin-top: auto;
  padding-top: var(--space-3);
  border-top: 1px solid var(--border-hairline);
}

.creator-card__count {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.creator-card__view {
  margin-left: auto;
  color: var(--tide);
  font-size: var(--text-sm);
}
</style>
