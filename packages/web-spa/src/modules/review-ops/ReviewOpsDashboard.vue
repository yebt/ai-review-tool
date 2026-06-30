<script setup lang="ts">
import { onMounted } from 'vue'

import ReviewCommentsList from './ReviewCommentsList.vue'
import ReviewCreateForm from './ReviewCreateForm.vue'
import ReviewDetailPanel from './ReviewDetailPanel.vue'
import ReviewEventLog from './ReviewEventLog.vue'
import ReviewHistoryList from './ReviewHistoryList.vue'
import type { CreateReviewRequest } from './types'
import { useReviewOps } from './useReviewOps'

const {
  history,
  selectedReview,
  comments,
  createState,
  selectedReviewId,
  events,
  eventConnectionStatus,
  selectedCommentsCount,
  refreshHistory,
  loadReview,
  submitReview,
} = useReviewOps()

onMounted(() => {
  void refreshHistory()
})

function handleSubmit(request: CreateReviewRequest) {
  void submitReview(request)
}

function handleSelect(reviewId: string) {
  void loadReview(reviewId)
}
</script>

<template>
  <section class="border-4 border-black bg-white p-5 shadow-[8px_8px_0_#000]">
    <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
      <div>
        <p class="font-mono text-sm font-bold uppercase tracking-[0.24em]">Phase 4.5 manual test surface</p>
        <h2 class="mt-2 text-3xl font-black uppercase tracking-[-0.04em] sm:text-5xl">
          Review operations
        </h2>
        <p class="mt-3 max-w-3xl font-bold text-zinc-700">
          Create a GitLab merge request review, inspect stored history, read generated comments, and watch the selected review's SSE stream.
        </p>
      </div>

      <div class="border-4 border-black bg-yellow-100 p-4 font-bold">
        <h3 class="font-black uppercase">Current limits</h3>
        <ul class="mt-2 list-square space-y-1 pl-5">
          <li>No publish or approval actions yet.</li>
          <li>No repo CRUD or memory setup yet.</li>
          <li>Provider/platform behavior depends on backend config and may be deterministic or fake.</li>
        </ul>
      </div>
    </div>

    <div class="mt-6 grid gap-6 xl:grid-cols-[minmax(18rem,0.75fr)_minmax(0,1.25fr)]">
      <div class="space-y-6">
        <ReviewCreateForm :state="createState" @submit-review="handleSubmit" />
        <ReviewHistoryList
          :state="history"
          :selected-review-id="selectedReviewId"
          @refresh="refreshHistory"
          @select-review="handleSelect"
        />
      </div>

      <div class="space-y-6">
        <ReviewDetailPanel :state="selectedReview" :comments-count="selectedCommentsCount" />
        <ReviewCommentsList :state="comments" />
        <ReviewEventLog :events="events" :connection-status="eventConnectionStatus" />
      </div>
    </div>
  </section>
</template>
