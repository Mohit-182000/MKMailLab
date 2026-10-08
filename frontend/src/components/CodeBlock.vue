<script setup lang="ts">
import CopyButton from './CopyButton.vue'

defineProps<{ code: string; language?: string }>()
</script>

<template>
  <div class="overflow-hidden rounded-md border border-border bg-code">
    <div class="flex h-8 items-center justify-between border-b border-border pr-1 pl-3">
      <span class="text-[11px] font-medium text-muted">{{ language }}</span>
      <CopyButton :text="code" label="Copy" with-text />
    </div>
    <pre class="overflow-x-auto p-3 font-mono text-xs leading-relaxed select-text"><code>{{ code }}</code></pre>
  </div>
</template>
