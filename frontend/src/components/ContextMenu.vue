<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch, type Component } from 'vue'

export interface MenuItem {
  label?: string
  icon?: Component
  shortcut?: string
  danger?: boolean
  divider?: boolean
  action?: () => void
}

const props = defineProps<{ open: boolean; x: number; y: number; items: MenuItem[] }>()
const emit = defineEmits<{ close: [] }>()

const menu = ref<HTMLElement | null>(null)
const pos = ref({ left: 0, top: 0 })

// Keep the menu inside the window.
watch(
  () => [props.open, props.x, props.y],
  async () => {
    if (!props.open) return
    pos.value = { left: props.x, top: props.y }
    await nextTick()
    const el = menu.value
    if (!el) return
    const r = el.getBoundingClientRect()
    pos.value = {
      left: Math.min(props.x, window.innerWidth - r.width - 8),
      top: Math.min(props.y, window.innerHeight - r.height - 8),
    }
    el.querySelector<HTMLButtonElement>('button')?.focus()
  },
)

function run(item: MenuItem) {
  emit('close')
  item.action?.()
}

function onDocDown(e: MouseEvent) {
  if (props.open && menu.value && !menu.value.contains(e.target as Node)) emit('close')
}
function onKey(e: KeyboardEvent) {
  if (!props.open) return
  if (e.key === 'Escape') emit('close')
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    const buttons = [...(menu.value?.querySelectorAll<HTMLButtonElement>('button') ?? [])]
    const i = buttons.indexOf(document.activeElement as HTMLButtonElement)
    const next = buttons[(i + (e.key === 'ArrowDown' ? 1 : -1) + buttons.length) % buttons.length]
    next?.focus()
  }
}
document.addEventListener('mousedown', onDocDown)
document.addEventListener('keydown', onKey)
window.addEventListener('blur', () => emit('close'))
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocDown)
  document.removeEventListener('keydown', onKey)
})
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      ref="menu"
      role="menu"
      class="fixed z-50 min-w-52 rounded-lg border border-border bg-panel p-1 shadow-pop"
      :style="{ left: `${pos.left}px`, top: `${pos.top}px` }"
      @contextmenu.prevent
    >
      <template v-for="(item, i) in items" :key="i">
        <div v-if="item.divider" class="my-1 h-px bg-border" role="separator" />
        <button
          v-else
          type="button"
          role="menuitem"
          class="flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-[12.5px] outline-none hover:bg-hover focus:bg-hover"
          :class="item.danger ? 'text-danger' : 'text-fg'"
          @click="run(item)"
        >
          <component :is="item.icon" v-if="item.icon" class="size-3.5 shrink-0" :class="item.danger ? '' : 'text-muted'" />
          <span v-else class="size-3.5 shrink-0" />
          <span class="flex-1">{{ item.label }}</span>
          <span v-if="item.shortcut" class="kbd">{{ item.shortcut }}</span>
        </button>
      </template>
    </div>
  </Teleport>
</template>
