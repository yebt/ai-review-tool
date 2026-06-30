<script setup lang="ts">
import { onMounted, shallowRef } from 'vue'
import { RefreshCw, Server } from '@lucide/vue'

import SmokeCard from './SmokeCard.vue'
import { useHealthCheck } from './useSmokeChecks'

const { health, loadHealth } = useHealthCheck()
const isRefreshing = shallowRef(false)

onMounted(() => {
  void refreshHealth()
})

function formatJson(value: unknown) {
  return JSON.stringify(value, null, 2)
}

async function refreshHealth() {
  isRefreshing.value = true

  try {
    await loadHealth()
  } finally {
    isRefreshing.value = false
  }
}
</script>

<template>
  <section class="space-y-6">
    <header class="border-4 border-black bg-white p-5 shadow-[8px_8px_0_#000]">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <p class="font-mono text-sm font-bold uppercase tracking-[0.24em]">GET /healthz</p>
          <h1 class="mt-2 text-4xl font-black uppercase tracking-[-0.04em] sm:text-6xl">
            Backend health
          </h1>
          <p class="mt-3 max-w-3xl font-bold text-zinc-700">
            Independent server liveness check. This page does not load skills or review data.
          </p>
        </div>
        <button
          type="button"
          class="inline-flex min-h-11 cursor-pointer items-center justify-center gap-2 border-4 border-black bg-yellow-300 px-5 py-3 font-black uppercase shadow-[4px_4px_0_#000] transition-colors hover:bg-lime-300 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-black disabled:cursor-not-allowed disabled:opacity-60"
          :disabled="isRefreshing"
          @click="refreshHealth"
        >
          <RefreshCw class="size-5" :class="{ 'animate-spin': isRefreshing }" aria-hidden="true" />
          Refresh health
        </button>
      </div>
    </header>

    <SmokeCard title="Server health" endpoint="GET /healthz" :status="health.status">
      <div v-if="health.status === 'idle'" class="font-bold">Health has not been requested yet.</div>
      <div v-else-if="health.status === 'loading'" class="font-bold">Checking server health...</div>
      <div v-else-if="health.status === 'error'" class="border-4 border-black bg-red-50 p-4 font-mono text-sm font-bold text-red-800">
        {{ health.error }}
      </div>
      <div v-else class="space-y-4">
        <div class="flex items-center gap-3">
          <Server class="size-7" aria-hidden="true" />
          <span class="text-2xl font-black uppercase">{{ health.data?.status ?? 'unknown' }}</span>
        </div>
        <pre class="overflow-auto border-4 border-black bg-black p-4 text-sm font-bold text-lime-300">{{ formatJson(health.data) }}</pre>
      </div>
    </SmokeCard>
  </section>
</template>
