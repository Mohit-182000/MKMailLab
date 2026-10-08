<script setup lang="ts">
import { CircleAlert, CircleCheck, Info, X } from 'lucide-vue-next'
import { useToastStore } from '@/stores/toast'

const toast = useToastStore()
</script>

<template>
  <div class="pointer-events-none fixed right-4 bottom-4 z-50 flex w-80 flex-col gap-2" aria-live="polite">
    <TransitionGroup
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="translate-y-2 opacity-0"
      leave-active-class="transition duration-100 ease-in"
      leave-to-class="opacity-0"
    >
      <div
        v-for="t in toast.toasts"
        :key="t.id"
        class="pointer-events-auto flex items-start gap-2.5 rounded-lg border border-border bg-panel p-3 shadow-pop"
        role="status"
      >
        <CircleCheck v-if="t.kind === 'success'" class="mt-px size-4 shrink-0 text-success" />
        <CircleAlert v-else-if="t.kind === 'error'" class="mt-px size-4 shrink-0 text-danger" />
        <Info v-else class="mt-px size-4 shrink-0 text-accent" />
        <div class="min-w-0 flex-1">
          <p class="font-medium">{{ t.title }}</p>
          <p v-if="t.message" class="mt-0.5 text-xs break-words text-muted select-text">{{ t.message }}</p>
        </div>
        <button class="text-faint hover:text-fg" aria-label="Dismiss" @click="toast.dismiss(t.id)">
          <X class="size-3.5" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>
