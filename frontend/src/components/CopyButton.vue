<script setup lang="ts">
import { Check, Copy } from 'lucide-vue-next'
import { useCopy } from '@/composables/useCopy'

const props = withDefaults(defineProps<{ text: string; label?: string; withText?: boolean }>(), {
  label: 'Copy',
  withText: false,
})

const { copy, copiedKey } = useCopy()
</script>

<template>
  <button
    type="button"
    :class="props.withText ? 'btn h-7 px-2 text-xs' : 'btn-icon size-7'"
    :title="props.label"
    :aria-label="props.label"
    @click="copy(props.text)"
  >
    <Check v-if="copiedKey" class="size-3.5 text-success" />
    <Copy v-else class="size-3.5" />
    <span v-if="props.withText">{{ copiedKey ? 'Copied' : props.label }}</span>
  </button>
</template>
