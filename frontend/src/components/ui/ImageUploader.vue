<script setup lang="ts">
import { computed, ref } from 'vue'
import AppIcon from '@/components/ui/AppIcon.vue'
import { uploadMedia } from '@/api/media'
import { errorMessage } from '@/components/features/format'
import type { Media } from '@/types'

const props = withDefaults(
  defineProps<{
    modelValue: string | null
    label: string
    hint?: string
    shape?: 'circle' | 'card' | 'wide'
    /** Rendered preview size in px; 'wide' ignores it and fills the row. */
    size?: number
    disabled?: boolean
  }>(),
  { hint: '', shape: 'card', size: 96, disabled: false },
)

const emit = defineEmits<{
  'update:modelValue': [value: string | null]
  /** The uploaded record, for callers that store the id rather than the URL. */
  uploaded: [media: Media]
}>()

const allowedTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']
const maxBytes = 5 * 1024 * 1024
const maxDimension = 6000

const input = ref<HTMLInputElement | null>(null)
const uploading = ref(false)
const dragging = ref(false)
const error = ref('')
const pickedName = ref('')
const pickedSize = ref('')

const hintId = computed(() => `upload-${props.label.toLowerCase().replace(/[^a-z0-9]+/g, '-')}-hint`)
const hasImage = computed(() => !!props.modelValue)
const previewStyle = computed(() =>
  props.shape === 'wide' ? undefined : { width: `${props.size}px`, height: `${props.size}px` },
)

function fail(message: string): void {
  error.value = message
}

function openPicker(): void {
  if (!props.disabled && !uploading.value) input.value?.click()
}

async function accept(file: File | null | undefined): Promise<void> {
  if (!file || props.disabled || uploading.value) return
  error.value = ''

  if (!allowedTypes.includes(file.type)) {
    fail('Choose a JPEG, PNG, GIF or WebP image.')
    return
  }
  if (file.size > maxBytes) {
    fail('Images must be 5 MB or smaller.')
    return
  }

  uploading.value = true
  try {
    const media = await uploadMedia(file)
    if (media.width > maxDimension || media.height > maxDimension) {
      fail('Images must be at most 6000 pixels on a side.')
      return
    }
    pickedName.value = file.name
    pickedSize.value = `${media.width} × ${media.height}`
    emit('update:modelValue', media.url)
    emit('uploaded', media)
  } catch (cause) {
    fail(errorMessage(cause, 'We could not upload that image. Try again.'))
  } finally {
    uploading.value = false
  }
}

function onPicked(event: Event): void {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  // Reset first so choosing the same file again still fires a change.
  target.value = ''
  void accept(file)
}

function onDrop(event: DragEvent): void {
  dragging.value = false
  void accept(event.dataTransfer?.files?.[0])
}

function remove(): void {
  error.value = ''
  pickedName.value = ''
  pickedSize.value = ''
  emit('update:modelValue', null)
}
</script>

<template>
  <div class="uploader" :class="[`uploader--${shape}`, { 'uploader--dragging': dragging }]">
    <span class="uploader__label">{{ label }}</span>

    <div class="uploader__body">
      <div
        class="uploader__preview"
        :style="previewStyle"
        @dragover.prevent="dragging = true"
        @dragleave="dragging = false"
        @drop.prevent="onDrop"
      >
        <img v-if="modelValue" :src="modelValue" alt="" />
        <span v-else class="uploader__placeholder" aria-hidden="true">
          <AppIcon name="creative" :size="22" />
        </span>
        <span v-if="uploading" class="uploader__busy" role="status">Uploading…</span>
      </div>

      <div class="uploader__controls">
        <div class="uploader__actions">
          <button
            type="button"
            class="uploader__button"
            :disabled="disabled || uploading"
            @click="openPicker"
          >
            {{ hasImage ? 'Replace image' : 'Upload image' }}
          </button>
          <button v-if="hasImage" type="button" class="uploader__remove" @click="remove">
            Remove
          </button>
        </div>

        <p class="uploader__meta">
          <span v-if="pickedName">{{ pickedName }}<template v-if="pickedSize"> · {{ pickedSize }}</template></span>
          <span v-else-if="hasImage">Current image</span>
          <span v-else>No image yet. Drop one here or choose a file.</span>
        </p>

        <p :id="hintId" class="uploader__hint">
          {{ hint || 'JPEG, PNG, GIF or WebP, up to 5 MB.' }}
        </p>

        <p v-if="error" class="uploader__error" role="alert">
          <AppIcon name="alert" :size="16" />
          <span>{{ error }}</span>
        </p>
      </div>
    </div>

    <input
      ref="input"
      class="uploader__input"
      type="file"
      :accept="allowedTypes.join(',')"
      :aria-label="label"
      :aria-describedby="hintId"
      tabindex="-1"
      @change="onPicked"
    />
  </div>
</template>

<style scoped>
.uploader {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  /* The uploader sits in rails and cards, so it reacts to its own width, not
     the viewport's. */
  container-type: inline-size;
}

.uploader__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.uploader__body {
  display: flex;
  align-items: flex-start;
  gap: var(--space-4);
}

/* A banner is the full width of whatever holds it: preview above controls. */
.uploader--wide .uploader__body {
  flex-direction: column;
}

.uploader__preview {
  position: relative;
  display: grid;
  flex: none;
  place-items: center;
  overflow: hidden;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  transition: border-color var(--dur-fast) var(--ease);
}

.uploader--wide .uploader__preview {
  width: 100%;
  min-height: 96px;
  aspect-ratio: 3 / 1;
}

.uploader--circle .uploader__preview {
  border-radius: var(--radius-pill);
}

.uploader--dragging .uploader__preview {
  border-color: var(--ember);
}

.uploader__preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.uploader__placeholder {
  display: grid;
  place-items: center;
  color: var(--text-secondary);
}

.uploader__busy {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: var(--scrim);
  color: var(--text-primary);
  font-size: var(--text-xs);
}

.uploader__controls {
  display: flex;
  min-width: 0;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--space-1);
}

.uploader__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
}

.uploader__button {
  min-height: 44px;
  padding: 0 var(--space-4);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-primary);
  font: inherit;
  font-weight: 500;
  cursor: pointer;
  transition:
    background-color var(--dur-fast) var(--ease),
    border-color var(--dur-fast) var(--ease);
}

.uploader__button:hover:not(:disabled) {
  border-color: var(--text-secondary);
  background: var(--ink-700);
}

.uploader__button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.uploader__remove {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  padding: 0 var(--space-2);
  border: 0;
  background: none;
  color: var(--tide);
  font: inherit;
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}

.uploader__remove:hover {
  color: var(--text-primary);
}

.uploader__meta {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  line-height: 1.45;
}

.uploader__hint {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.uploader__error {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--alert);
  font-size: var(--text-sm);
}

/* Focusable for assistive tech, invisible on screen: the button is the affordance. */
.uploader__input {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip-path: inset(50%);
  border: 0;
  white-space: nowrap;
}

/* Narrow container (a rail card, a phone): stack instead of squeezing the
   controls into whatever the preview leaves. */
@container (max-width: 420px) {
  .uploader__body {
    flex-direction: column;
    gap: var(--space-3);
  }

  .uploader__actions {
    width: 100%;
  }
}
</style>
