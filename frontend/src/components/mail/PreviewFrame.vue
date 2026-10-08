<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { MonitorSmartphone } from 'lucide-vue-next'
import { DEVICE_PRESETS, clampWidth, frameLayout } from '@/utils/devices'

const props = defineProps<{ src: string; messageId: number }>()

const STORAGE_KEY = 'lm.preview.device'
const CUSTOM = 'custom'

function readStored(): { id: string; custom: number } {
  try {
    const v = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '{}') as { id?: string; custom?: number }
    return { id: v.id ?? 'fit', custom: v.custom ?? 800 }
  } catch {
    return { id: 'fit', custom: 800 }
  }
}

const stored = readStored()
const deviceId = ref(stored.id)
const customWidth = ref(stored.custom)

watch([deviceId, customWidth], () => {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ id: deviceId.value, custom: customWidth.value }))
  } catch {
    // not persisted
  }
})

const groups = computed(() => {
  const out: { name: string; items: typeof DEVICE_PRESETS }[] = []
  for (const p of DEVICE_PRESETS) {
    const g = out.find((x) => x.name === p.group)
    if (g) g.items.push(p)
    else out.push({ name: p.group, items: [p] })
  }
  return out
})

const targetWidth = computed(() => {
  if (deviceId.value === CUSTOM) return clampWidth(customWidth.value)
  return DEVICE_PRESETS.find((p) => p.id === deviceId.value)?.width ?? 0
})

// Track the available area so wide viewports can be scaled to fit.
const stage = ref<HTMLElement | null>(null)
const avail = ref({ w: 0, h: 0 })
let observer: ResizeObserver | null = null

onMounted(() => {
  observer = new ResizeObserver(([entry]) => {
    if (!entry) return
    avail.value = { w: entry.contentRect.width, h: entry.contentRect.height }
  })
  if (stage.value) observer.observe(stage.value)
})
onBeforeUnmount(() => observer?.disconnect())

const layout = computed(() => frameLayout(targetWidth.value, avail.value.w, avail.value.h))
const scalePct = computed(() => Math.round(layout.value.scale * 100))

const quick = [
  { id: 'fit', label: 'Fit' },
  { id: 'desktop', label: 'Desktop' },
  { id: 'ipad-air', label: 'Tablet' },
  { id: 'iphone', label: 'Mobile' },
]

function setCustom(e: Event) {
  const v = Number((e.target as HTMLInputElement).value)
  if (v > 0) {
    customWidth.value = clampWidth(v)
    deviceId.value = CUSTOM
  }
}

</script>

<template>
  <div class="flex h-full flex-col">
    <div class="flex h-10 shrink-0 items-center gap-2 border-b border-border bg-panel px-3">
      <div class="flex items-center rounded-md bg-panel-2 p-0.5">
        <button
          v-for="q in quick"
          :key="q.id"
          type="button"
          class="h-6 rounded px-2 text-xs font-medium transition-colors"
          :class="deviceId === q.id ? 'bg-panel text-fg shadow-sm' : 'text-muted hover:text-fg'"
          @click="deviceId = q.id"
        >
          {{ q.label }}
        </button>
      </div>

      <label class="relative flex items-center">
        <MonitorSmartphone class="pointer-events-none absolute left-2 size-3.5 text-muted" />
        <select
          v-model="deviceId"
          class="h-7 appearance-none rounded-md border border-border bg-panel pr-6 pl-7 text-xs text-fg outline-none focus:border-accent"
          aria-label="Preview screen size"
        >
          <optgroup v-for="g in groups" :key="g.name" :label="g.name">
            <option v-for="p in g.items" :key="p.id" :value="p.id">
              {{ p.label }}{{ p.width ? ` · ${p.width}px` : '' }}
            </option>
          </optgroup>
          <optgroup label="Custom">
            <option :value="CUSTOM">Custom width…</option>
          </optgroup>
        </select>
        <span class="pointer-events-none absolute right-2 text-[9px] text-muted">▼</span>
      </label>

      <label class="flex items-center gap-1 text-xs text-muted">
        <input
          type="number"
          :min="240"
          :max="3840"
          step="10"
          class="h-7 w-20 rounded-md border border-border bg-panel px-2 text-right font-mono text-xs text-fg outline-none focus:border-accent"
          :value="targetWidth || ''"
          :placeholder="String(Math.round(avail.w))"
          aria-label="Custom preview width in pixels"
          @change="setCustom"
        />
        px
      </label>

      <span class="ml-auto flex items-center gap-2 text-[11px] text-faint">
        <span v-if="layout.scale < 1" class="rounded bg-panel-2 px-1.5 py-0.5 text-muted" title="Rendered at full width, scaled to fit the panel">
          {{ scalePct }}%
        </span>
        <span>Scripts disabled</span>
      </span>
    </div>

    <div ref="stage" class="relative min-h-0 flex-1 overflow-hidden" :class="targetWidth ? 'bg-panel-2' : ''">
      <!-- Wide viewports render at true width (media queries stay correct) and are scaled down. -->
      <div
        class="absolute top-0 left-1/2 origin-top"
        :style="{
          width: layout.width ? `${layout.width}px` : '100%',
          height: `${layout.height}px`,
          transform: `translateX(-50%) scale(${layout.scale})`,
        }"
      >
        <iframe
          :key="messageId"
          :src="src"
          sandbox=""
          referrerpolicy="no-referrer"
          title="Email preview"
          class="block size-full bg-white"
          :class="targetWidth ? 'border-x border-border shadow-pop' : ''"
        />
      </div>
    </div>
  </div>
</template>
