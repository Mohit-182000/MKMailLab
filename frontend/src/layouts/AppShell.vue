<script setup lang="ts">
import { computed } from 'vue'
import { Inbox, Monitor, Moon, Settings2, Sun } from 'lucide-vue-next'
import SmtpToggle from '@/components/SmtpToggle.vue'
import ToastHost from '@/components/ToastHost.vue'
import { useTheme } from '@/composables/useTheme'
import { useMailStore } from '@/stores/mail'

const mail = useMailStore()
const theme = useTheme()

const tabs = computed(() => [
  { to: '/inbox', label: 'Inbox', icon: Inbox, badge: mail.inboxUnread },
  { to: '/config', label: 'Configuration', icon: Settings2, badge: 0 },
])

const themeIcon = computed(() => ({ system: Monitor, light: Sun, dark: Moon })[theme.preference.value])
const themeLabel = computed(() => `Theme: ${theme.preference.value} (click to change)`)
</script>

<template>
  <div class="flex h-full flex-col">
    <header class="flex h-13 shrink-0 items-center gap-6 border-b border-border bg-panel px-4">
      <div class="flex items-center gap-2.5">
        <div class="flex size-7 items-center justify-center rounded-md bg-accent text-accent-fg">
          <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <rect x="3" y="5" width="18" height="14" rx="2.5" />
            <path d="m4 7 8 6 8-6" />
          </svg>
        </div>
        <span class="text-[14px] font-semibold tracking-tight">LocalMail</span>
      </div>

      <nav class="flex h-full items-stretch gap-1" aria-label="Main">
        <RouterLink
          v-for="tab in tabs"
          :key="tab.to"
          :to="tab.to"
          class="group relative flex items-center gap-2 px-3 text-muted transition-colors hover:text-fg [&.is-active]:text-fg"
          active-class="is-active"
        >
          <component :is="tab.icon" class="size-4" />
          <span class="font-medium">{{ tab.label }}</span>
          <span
            v-if="tab.badge > 0"
            class="min-w-5 rounded-full bg-accent px-1.5 text-center text-[10px] leading-[18px] font-semibold text-accent-fg"
          >
            {{ tab.badge > 999 ? '999+' : tab.badge }}
          </span>
          <span class="absolute inset-x-2 -bottom-px h-0.5 rounded-full bg-accent opacity-0 transition-opacity group-[.is-active]:opacity-100" />
        </RouterLink>
      </nav>

      <div class="ml-auto flex items-center gap-2">
        <SmtpToggle />
        <button class="btn-icon" :title="themeLabel" :aria-label="themeLabel" @click="theme.cycle()">
          <component :is="themeIcon" class="size-4" />
        </button>
      </div>
    </header>

    <main class="min-h-0 flex-1">
      <RouterView />
    </main>

    <ToastHost />
  </div>
</template>
