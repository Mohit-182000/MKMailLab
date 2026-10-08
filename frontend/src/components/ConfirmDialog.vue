<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    message?: string
    confirmLabel?: string
    danger?: boolean
  }>(),
  { confirmLabel: 'Confirm', danger: false },
)
const emit = defineEmits<{ confirm: []; cancel: [] }>()
const confirmBtn = ref<HTMLButtonElement | null>(null)

watch(
  () => props.open,
  async (open) => {
    if (open) {
      await nextTick()
      confirmBtn.value?.focus()
    }
  },
)
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-150"
      enter-from-class="opacity-0"
      leave-active-class="transition duration-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="open"
        class="fixed inset-0 z-40 flex items-center justify-center bg-black/40 p-4"
        @click.self="emit('cancel')"
        @keydown.esc="emit('cancel')"
      >
        <div role="alertdialog" aria-modal="true" :aria-label="title" class="w-full max-w-sm rounded-lg border border-border bg-panel p-5 shadow-pop">
          <h2 class="text-sm font-semibold">{{ title }}</h2>
          <p v-if="message" class="mt-1.5 text-muted">{{ message }}</p>
          <div class="mt-5 flex justify-end gap-2">
            <button class="btn" @click="emit('cancel')">Cancel</button>
            <button ref="confirmBtn" class="btn" :class="danger ? 'btn-danger' : 'btn-primary'" @click="emit('confirm')">
              {{ confirmLabel }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
