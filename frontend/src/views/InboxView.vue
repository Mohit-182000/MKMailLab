<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import MessageList from '@/components/mail/MessageList.vue'
import MessageView from '@/components/mail/MessageView.vue'
import { useMailStore } from '@/stores/mail'

const mail = useMailStore()
const list = ref<InstanceType<typeof MessageList> | null>(null)

function isTyping(e: KeyboardEvent): boolean {
  const el = e.target as HTMLElement | null
  return !!el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable)
}

function onKey(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    list.value?.focusSearch()
    return
  }
  if (isTyping(e) || e.ctrlKey || e.metaKey || e.altKey) return
  const id = mail.selectedId
  switch (e.key) {
    case 'ArrowDown':
    case 'j':
      e.preventDefault()
      if (id === null && mail.items[0]) void mail.select(mail.items[0].id)
      else mail.selectAdjacent(1)
      break
    case 'ArrowUp':
    case 'k':
      e.preventDefault()
      mail.selectAdjacent(-1)
      break
    case 'Escape':
      void mail.select(null)
      break
    case 'Delete':
      if (id !== null) void mail.remove(id)
      break
    case 'r':
    case 'R':
      void mail.load()
      break
    case 's':
    case 'S':
      if (id !== null) void mail.toggleStar(id)
      break
    case 'u':
    case 'U':
      if (id !== null) void mail.setRead(id, false)
      break
  }
}

onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <div class="grid h-full grid-cols-[minmax(300px,380px)_1fr] max-xl:grid-cols-[320px_1fr]">
    <MessageList ref="list" />
    <MessageView />
  </div>
</template>
