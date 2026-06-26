<script setup lang="ts">
import StatusBadge from './StatusBadge.vue'
import type { CheckStatus } from './types'

defineProps<{
  title: string
  endpoint: string
  status: CheckStatus
}>()
</script>

<template>
  <section class="border-4 border-black bg-white p-5 shadow-[8px_8px_0_#000]">
    <div class="flex flex-col gap-3 border-b-4 border-black pb-4 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <p class="font-mono text-xs font-bold uppercase tracking-[0.2em] text-zinc-600">{{ endpoint }}</p>
        <h2 class="mt-1 text-2xl font-black uppercase tracking-tight">{{ title }}</h2>
      </div>
      <StatusBadge :status="status" />
    </div>

    <div class="pt-5">
      <slot />
    </div>
  </section>
</template>
