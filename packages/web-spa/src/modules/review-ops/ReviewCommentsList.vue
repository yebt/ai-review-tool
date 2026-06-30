<script setup lang="ts">
import type { AsyncState, ReviewComment } from './types'

defineProps<{
  state: Readonly<AsyncState<readonly ReviewComment[]>>
}>()
</script>

<template>
  <section class="border-4 border-black bg-zinc-50 p-4">
    <p class="font-mono text-xs font-black uppercase tracking-[0.2em]">GET /api/v1/reviews/:id/comments</p>
    <h3 class="mt-1 text-2xl font-black uppercase">Generated comments</h3>
    <p class="mt-2 font-bold text-zinc-700">Read-only in Phase 4.5. Approval and publish controls arrive later.</p>

    <div v-if="state.status === 'idle'" class="mt-4 font-bold">Select a review to load comments.</div>
    <div v-else-if="state.status === 'loading'" class="mt-4 font-bold">Loading comments...</div>
    <div v-else-if="state.status === 'error'" class="mt-4 border-4 border-black bg-red-50 p-3 font-mono text-sm font-bold text-red-800">
      {{ state.error }}
    </div>
    <div v-else-if="!state.data?.length" class="mt-4 border-4 border-black bg-white p-4 font-bold">
      This review has no generated comments.
    </div>

    <div v-else class="mt-4 grid gap-4">
      <article v-for="comment in state.data" :key="comment.id" class="border-4 border-black bg-white p-4">
        <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
          <div>
            <h4 class="text-xl font-black uppercase">{{ comment.dimension }} / {{ comment.severity }}</h4>
            <p class="mt-1 break-all font-mono text-sm font-bold">{{ comment.file }}:{{ comment.line_start || '?' }}</p>
          </div>
          <span class="w-fit border-2 border-black bg-yellow-100 px-2 py-1 font-mono text-xs font-black uppercase">{{ comment.status }}</span>
        </div>

        <dl class="mt-4 space-y-3">
          <div>
            <dt class="font-mono text-xs font-black uppercase">Evidence</dt>
            <dd class="mt-1 font-bold">{{ comment.evidence }}</dd>
          </div>
          <div>
            <dt class="font-mono text-xs font-black uppercase">Why</dt>
            <dd class="mt-1 font-bold">{{ comment.why }}</dd>
          </div>
          <div v-if="comment.suggestion_snippet">
            <dt class="font-mono text-xs font-black uppercase">Suggestion</dt>
            <dd class="mt-1 whitespace-pre-wrap border-2 border-black bg-zinc-100 p-3 font-mono text-sm font-bold">{{ comment.suggestion_snippet }}</dd>
          </div>
        </dl>
      </article>
    </div>
  </section>
</template>
