<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Info, TriangleAlert } from 'lucide-vue-next'
import CodeBlock from '@/components/CodeBlock.vue'
import TechPicker from './TechPicker.vue'
import { useSmtpStore } from '@/stores/smtp'
import { buildSnippets } from '@/utils/snippets'

const STORAGE_KEY = 'lm.snippet'
const QUICK = ['laravel', 'node', 'python', 'dotnet', 'spring', 'go']

const smtp = useSmtpStore()

function readStored(): string {
  try {
    return localStorage.getItem(STORAGE_KEY) || 'laravel'
  } catch {
    return 'laravel'
  }
}

const active = ref(readStored())
watch(active, (id) => {
  try {
    localStorage.setItem(STORAGE_KEY, id)
  } catch {
    // storage unavailable; selection just isn't remembered
  }
})

const snippets = computed(() => (smtp.connection ? buildSnippets(smtp.connection) : []))
const current = computed(() => snippets.value.find((s) => s.id === active.value) ?? snippets.value[0])
const quick = computed(() => QUICK.map((id) => snippets.value.find((s) => s.id === id)).filter((s) => !!s))
const authConflict = computed(() => !!current.value?.noAuth && !!smtp.connection?.authRequired)
</script>

<template>
  <section class="card">
    <header class="border-b border-border px-5 py-3.5">
      <h2 class="text-sm font-semibold">Framework setup</h2>
      <p class="mt-0.5 text-xs text-muted">
        Copy-paste configuration for {{ snippets.length }} languages, frameworks and tools, kept in sync with your settings.
      </p>
    </header>

    <div class="space-y-3 p-4">
      <TechPicker v-model="active" :snippets="snippets" />

      <div class="flex flex-wrap items-center gap-1.5">
        <span class="text-[11px] text-faint">Popular:</span>
        <button
          v-for="s in quick"
          :key="s.id"
          type="button"
          class="h-6 rounded-full border px-2.5 text-[11px] font-medium transition-colors"
          :class="s.id === current?.id ? 'border-accent bg-accent-soft text-fg' : 'border-border text-muted hover:bg-hover hover:text-fg'"
          @click="active = s.id"
        >
          {{ s.label.replace(/ \(.*\)$/, '') }}
        </button>
      </div>

      <template v-if="current">
        <p v-if="authConflict" class="flex gap-2 rounded-md border border-warning/30 bg-warning-soft px-3 py-2 text-xs">
          <TriangleAlert class="mt-px size-3.5 shrink-0 text-warning" />
          This integration cannot send a username and password, but your server requires authentication. Switch Authentication to "Accept any" in Server settings.
        </p>
        <p v-if="current.note" class="flex gap-2 rounded-md bg-panel-2 px-3 py-2 text-xs text-muted">
          <Info class="mt-px size-3.5 shrink-0" /> <span class="select-text">{{ current.note }}</span>
        </p>
        <CodeBlock v-if="current.install" :code="current.install" language="Install" />
        <CodeBlock :code="current.code" :language="current.language" />
      </template>
    </div>
  </section>
</template>
