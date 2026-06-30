<script setup lang="ts">
import type { EventConnectionStatus, ReviewEvent } from './types'

defineProps<{
  events: readonly ReviewEvent[]
  connectionStatus: EventConnectionStatus
}>()

function formatJson(value: unknown) {
  return JSON.stringify(value, null, 2)
}
</script>

<template>
  <section class="border-4 border-black bg-zinc-50 p-4">
    <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
      <div>
        <p class="font-mono text-xs font-black uppercase tracking-[0.2em]">GET /api/v1/reviews/:id/events</p>
        <h3 class="mt-1 text-2xl font-black uppercase">SSE event log</h3>
      </div>
      <span class="w-fit border-4 border-black bg-white px-3 py-2 font-mono text-sm font-black uppercase">{{ connectionStatus }}</span>
    </div>

    <div v-if="events.length === 0" class="mt-4 font-bold">
      Select a review to open an EventSource stream. Existing reviews may not replay old events.
    </div>

    <ol v-else class="mt-4 grid gap-3">
      <li v-for="event in events" :key="event.id" class="border-4 border-black bg-white p-3">
        <div class="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
          <span class="font-black uppercase">{{ event.name }}</span>
          <time class="font-mono text-xs font-bold text-zinc-700">{{ event.receivedAt }}</time>
        </div>
        <pre class="mt-3 overflow-auto bg-black p-3 font-mono text-sm font-bold text-lime-300">{{ formatJson(event.data) }}</pre>
      </li>
    </ol>
  </section>
</template>
