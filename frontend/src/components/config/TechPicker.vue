<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { Check, ChevronsUpDown, Search } from 'lucide-vue-next'
import { SNIPPET_GROUPS, type Snippet } from '@/utils/snippets'

const props = defineProps<{ snippets: Snippet[]; modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [id: string] }>()

const open = ref(false)
const query = ref('')
const active = ref(0)
const root = ref<HTMLElement | null>(null)
const input = ref<HTMLInputElement | null>(null)
const listEl = ref<HTMLElement | null>(null)

const selected = computed(() => props.snippets.find((s) => s.id === props.modelValue))

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return props.snippets
  return props.snippets.filter((s) =>
    [s.label, s.group, s.language, ...(s.keywords ?? [])].some((t) => t.toLowerCase().includes(q)),
  )
})

/** Filtered snippets grouped in display order; flat index drives keyboard nav. */
const groups = computed(() => {
  let index = 0
  return SNIPPET_GROUPS.map((g) => ({
    name: g,
    items: filtered.value.filter((s) => s.group === g).map((s) => ({ snippet: s, index: index++ })),
  })).filter((g) => g.items.length > 0)
})
const flat = computed(() => groups.value.flatMap((g) => g.items.map((i) => i.snippet)))

watch(query, () => (active.value = 0))

async function toggle() {
  if (open.value) return close()
  open.value = true
  query.value = ''
  active.value = Math.max(0, flat.value.findIndex((s) => s.id === props.modelValue))
  await nextTick()
  input.value?.focus()
  scrollActive()
}

function close() {
  open.value = false
}

function choose(s: Snippet) {
  emit('update:modelValue', s.id)
  close()
}

function scrollActive() {
  nextTick(() => listEl.value?.querySelector(`[data-index="${active.value}"]`)?.scrollIntoView({ block: 'nearest' }))
}

function onKey(e: KeyboardEvent) {
  const n = flat.value.length
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    if (n) active.value = (active.value + 1) % n
    scrollActive()
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    if (n) active.value = (active.value - 1 + n) % n
    scrollActive()
  } else if (e.key === 'Enter') {
    e.preventDefault()
    const s = flat.value[active.value]
    if (s) choose(s)
  } else if (e.key === 'Escape') {
    e.preventDefault()
    close()
  }
}

function onDocClick(e: MouseEvent) {
  if (open.value && root.value && !root.value.contains(e.target as Node)) close()
}
document.addEventListener('mousedown', onDocClick)
onBeforeUnmount(() => document.removeEventListener('mousedown', onDocClick))
</script>

<template>
  <div ref="root" class="relative">
    <button
      type="button"
      class="flex h-9 w-full items-center gap-2 rounded-md border border-border bg-panel px-3 text-left transition-colors hover:bg-hover"
      :class="open ? 'border-accent ring-2 ring-accent/20' : ''"
      aria-haspopup="listbox"
      :aria-expanded="open"
      @click="toggle"
    >
      <span class="min-w-0 flex-1 truncate">
        <span class="font-medium">{{ selected?.label ?? 'Choose a language or framework' }}</span>
        <span v-if="selected" class="ml-2 text-xs text-muted">{{ selected.group }}</span>
      </span>
      <span class="text-[11px] text-faint">{{ snippets.length }} options</span>
      <ChevronsUpDown class="size-3.5 shrink-0 text-muted" />
    </button>

    <Transition
      enter-active-class="transition duration-100 ease-out"
      enter-from-class="-translate-y-1 opacity-0"
      leave-active-class="transition duration-75"
      leave-to-class="opacity-0"
    >
      <div
        v-if="open"
        class="absolute inset-x-0 top-full z-30 mt-1 overflow-hidden rounded-lg border border-border bg-panel shadow-pop"
      >
        <div class="relative border-b border-border">
          <Search class="pointer-events-none absolute top-1/2 left-3 size-3.5 -translate-y-1/2 text-faint" />
          <input
            ref="input"
            v-model="query"
            class="h-10 w-full bg-transparent pr-3 pl-9 text-[13px] outline-none placeholder:text-faint"
            placeholder="Search languages, frameworks, tools…"
            role="combobox"
            aria-autocomplete="list"
            aria-controls="tech-list"
            :aria-activedescendant="flat[active] ? `tech-${flat[active]!.id}` : undefined"
            @keydown="onKey"
          />
        </div>
        <div id="tech-list" ref="listEl" class="max-h-80 overflow-y-auto p-1" role="listbox">
          <p v-if="groups.length === 0" class="px-3 py-6 text-center text-xs text-muted">
            No match. Use <button class="text-accent underline" @click="choose(snippets.find((s) => s.id === 'generic')!)">Any SMTP client</button> settings.
          </p>
          <div v-for="g in groups" :key="g.name" role="group" :aria-label="g.name">
            <p class="px-2.5 pt-2 pb-1 text-[10px] font-semibold tracking-wider text-faint uppercase">{{ g.name }}</p>
            <button
              v-for="item in g.items"
              :id="`tech-${item.snippet.id}`"
              :key="item.snippet.id"
              type="button"
              role="option"
              :data-index="item.index"
              :aria-selected="item.snippet.id === modelValue"
              class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left"
              :class="item.index === active ? 'bg-active' : 'hover:bg-hover'"
              @mousemove="active = item.index"
              @click="choose(item.snippet)"
            >
              <span class="flex-1 truncate">{{ item.snippet.label }}</span>
              <span class="truncate text-[11px] text-faint">{{ item.snippet.language }}</span>
              <Check v-if="item.snippet.id === modelValue" class="size-3.5 shrink-0 text-accent" />
              <span v-else class="w-3.5 shrink-0" />
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>
