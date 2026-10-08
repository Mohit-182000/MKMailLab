<script setup lang="ts">
import { computed, ref } from 'vue'
import CodeBlock from '@/components/CodeBlock.vue'
import { useSmtpStore } from '@/stores/smtp'
import { buildSnippets } from '@/utils/snippets'

const smtp = useSmtpStore()
const active = ref('laravel')

const snippets = computed(() => (smtp.connection ? buildSnippets(smtp.connection) : []))
const current = computed(() => snippets.value.find((s) => s.id === active.value) ?? snippets.value[0])
</script>

<template>
  <section class="card">
    <header class="border-b border-border px-5 pt-3.5">
      <h2 class="text-sm font-semibold">Framework setup</h2>
      <p class="mt-0.5 text-xs text-muted">Copy-paste configuration, kept in sync with your settings.</p>
      <nav class="mt-3 -mb-px flex gap-1 overflow-x-auto" role="tablist">
        <button
          v-for="s in snippets"
          :key="s.id"
          role="tab"
          :aria-selected="active === s.id"
          class="relative h-8 shrink-0 px-2.5 text-xs font-medium transition-colors"
          :class="active === s.id ? 'text-fg' : 'text-muted hover:text-fg'"
          @click="active = s.id"
        >
          {{ s.label }}
          <span v-if="active === s.id" class="absolute inset-x-1.5 bottom-0 h-0.5 rounded-full bg-accent" />
        </button>
      </nav>
    </header>
    <div class="p-4">
      <CodeBlock v-if="current" :code="current.code" :language="current.language" />
    </div>
  </section>
</template>
