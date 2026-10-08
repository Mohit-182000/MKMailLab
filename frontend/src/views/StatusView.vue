<script setup lang="ts">
// Phase 1 placeholder: confirms the Go ↔ Vue bridge and data paths.
// Replaced by the AppShell and Dashboard in Phase 2.
import { computed } from 'vue'
import { useAppStore } from '@/stores/app'

const app = useAppStore()

const rows = computed(() => {
  const i = app.info
  if (!i) return []
  return [
    { label: 'Version', value: `${i.version}${i.dev ? ' (dev)' : ''}` },
    { label: 'Mode', value: i.portable ? 'Portable' : 'Installed' },
    { label: 'Data', value: i.dataDir },
    { label: 'Database', value: i.databasePath },
    { label: 'Log file', value: i.logFile },
    { label: 'Runtime', value: `${i.goVersion} · ${i.platform}` },
  ]
})
</script>

<template>
  <main class="flex h-full items-center justify-center p-8">
    <section
      class="w-full max-w-xl rounded-lg border border-border bg-surface p-6"
      aria-labelledby="status-title"
    >
      <header class="mb-5 flex items-center gap-3">
        <div class="flex size-8 items-center justify-center rounded-md bg-accent text-sm font-semibold text-white">
          L
        </div>
        <div>
          <h1 id="status-title" class="text-base font-semibold">LocalMail</h1>
          <p class="text-muted">Backend foundation</p>
        </div>
        <span
          class="ml-auto inline-flex items-center gap-1.5 rounded-full border border-border px-2.5 py-0.5 text-xs"
          role="status"
        >
          <span
            class="size-1.5 rounded-full"
            :class="{
              'bg-emerald-500': app.state === 'ready',
              'bg-amber-500 animate-pulse': app.state === 'loading' || app.state === 'idle',
              'bg-red-500': app.state === 'error',
            }"
          />
          {{ app.state === 'ready' ? 'Connected' : app.state === 'error' ? 'Error' : 'Connecting…' }}
        </span>
      </header>

      <p v-if="app.error" class="rounded-md border border-red-500/30 bg-red-500/10 p-3 text-red-600 dark:text-red-400">
        {{ app.error }}
      </p>

      <dl v-else class="grid grid-cols-[7rem_1fr] gap-x-4 gap-y-2">
        <template v-for="row in rows" :key="row.label">
          <dt class="text-muted">{{ row.label }}</dt>
          <dd class="truncate font-mono text-xs leading-5 select-text" :title="row.value">{{ row.value }}</dd>
        </template>
      </dl>
    </section>
  </main>
</template>
