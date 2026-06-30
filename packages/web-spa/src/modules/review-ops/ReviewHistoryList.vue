<script setup lang="ts">
import { RefreshCw } from '@lucide/vue'

import type { AsyncState, ReviewRecord } from './types'

defineProps<{
  state: Readonly<AsyncState<readonly ReviewRecord[]>>
  selectedReviewId: string | null
}>()

defineEmits<{
  refresh: []
  selectReview: [reviewId: string]
}>()

function reviewLabel(review: ReviewRecord) {
  return review.mr_title || `${review.project_path || review.project_url || 'GitLab project'} !${review.mr_id || 'unknown'}`
}
</script>

<template>
  <section class="border-4 border-black bg-zinc-50 p-4">
    <div class="flex items-center justify-between gap-3">
      <div>
        <p class="font-mono text-xs font-black uppercase tracking-[0.2em]">GET /api/v1/reviews</p>
        <h3 class="mt-1 text-2xl font-black uppercase">Review history</h3>
      </div>
      <button
        type="button"
        class="inline-flex min-h-11 cursor-pointer items-center justify-center border-4 border-black bg-white p-2 shadow-[3px_3px_0_#000] hover:bg-yellow-200 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-black"
        :disabled="state.status === 'loading'"
        aria-label="Refresh review history"
        @click="$emit('refresh')"
      >
        <RefreshCw class="size-5" :class="{ 'animate-spin': state.status === 'loading' }" aria-hidden="true" />
      </button>
    </div>

    <div v-if="state.status === 'idle'" class="mt-4 font-bold">History has not loaded yet.</div>
    <div v-else-if="state.status === 'loading'" class="mt-4 font-bold">Loading reviews...</div>
    <div v-else-if="state.status === 'error'" class="mt-4 border-4 border-black bg-red-50 p-3 font-mono text-sm font-bold text-red-800">
      {{ state.error }}
    </div>
    <div v-else-if="!state.data?.length" class="mt-4 font-bold">No reviews have been stored yet.</div>

    <div v-else class="mt-4 grid gap-3">
      <button
        v-for="review in state.data"
        :key="review.id"
        type="button"
        data-testid="review-history-item"
        class="cursor-pointer border-4 border-black p-3 text-left transition-colors hover:bg-yellow-100 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-black"
        :class="review.id === selectedReviewId ? 'bg-lime-200 shadow-[4px_4px_0_#000]' : 'bg-white'"
        @click="$emit('selectReview', review.id)"
      >
        <div class="flex items-start justify-between gap-3">
          <span class="font-black uppercase">{{ reviewLabel(review) }}</span>
          <span class="border-2 border-black bg-white px-2 py-1 font-mono text-xs font-black uppercase">{{ review.status }}</span>
        </div>
        <dl class="mt-3 grid gap-2 font-mono text-xs text-zinc-700">
          <div><dt class="inline font-black uppercase">ID:</dt> <dd class="inline break-all">{{ review.id }}</dd></div>
          <div><dt class="inline font-black uppercase">MR:</dt> <dd class="inline">!{{ review.mr_id || 'n/a' }}</dd></div>
          <div><dt class="inline font-black uppercase">Verdict:</dt> <dd class="inline">{{ review.verdict || 'pending' }}</dd></div>
        </dl>
      </button>
    </div>
  </section>
</template>
