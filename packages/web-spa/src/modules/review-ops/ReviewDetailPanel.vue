<script setup lang="ts">
import type { AsyncState, ReviewRecord } from './types'

defineProps<{
  state: Readonly<AsyncState<ReviewRecord>>
  commentsCount: number
}>()

function formatJson(value: unknown) {
  return JSON.stringify(value, null, 2)
}
</script>

<template>
  <section class="border-4 border-black bg-zinc-50 p-4">
    <p class="font-mono text-xs font-black uppercase tracking-[0.2em]">GET /api/v1/reviews/:id</p>
    <h3 class="mt-1 text-2xl font-black uppercase">Review detail</h3>

    <div v-if="state.status === 'idle'" class="mt-4 font-bold">Select a review to inspect the persisted detail.</div>
    <div v-else-if="state.status === 'loading'" class="mt-4 font-bold">Loading review detail...</div>
    <div v-else-if="state.status === 'error'" class="mt-4 border-4 border-black bg-red-50 p-3 font-mono text-sm font-bold text-red-800">
      {{ state.error }}
    </div>

    <article v-else-if="state.data" class="mt-4 space-y-4">
      <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div>
          <h4 class="text-xl font-black uppercase">{{ state.data.mr_title || 'Untitled merge request' }}</h4>
          <p class="mt-1 break-all font-mono text-sm font-bold text-zinc-700">{{ state.data.id }}</p>
        </div>
        <span class="w-fit border-4 border-black bg-lime-200 px-3 py-2 font-mono text-sm font-black uppercase">{{ state.data.status }}</span>
      </div>

      <dl class="grid gap-3 md:grid-cols-2">
        <div class="border-2 border-black bg-white p-3">
          <dt class="font-mono text-xs font-black uppercase">Project</dt>
          <dd class="break-all font-bold">{{ state.data.project_path || state.data.project_url || 'n/a' }}</dd>
        </div>
        <div class="border-2 border-black bg-white p-3">
          <dt class="font-mono text-xs font-black uppercase">MR IID</dt>
          <dd class="font-bold">!{{ state.data.mr_id || 'n/a' }}</dd>
        </div>
        <div class="border-2 border-black bg-white p-3">
          <dt class="font-mono text-xs font-black uppercase">Verdict</dt>
          <dd class="font-bold uppercase">{{ state.data.verdict || 'pending' }}</dd>
        </div>
        <div class="border-2 border-black bg-white p-3">
          <dt class="font-mono text-xs font-black uppercase">Generated comments</dt>
          <dd class="font-bold">{{ commentsCount }}</dd>
        </div>
        <div class="border-2 border-black bg-white p-3">
          <dt class="font-mono text-xs font-black uppercase">Model/provider</dt>
          <dd class="font-bold">{{ state.data.model_used || 'n/a' }}</dd>
        </div>
        <div class="border-2 border-black bg-white p-3">
          <dt class="font-mono text-xs font-black uppercase">Completed</dt>
          <dd class="font-bold">{{ state.data.completed_at || 'not complete' }}</dd>
        </div>
      </dl>

      <div v-if="state.data.scores" class="border-2 border-black bg-white p-3">
        <h5 class="font-mono text-xs font-black uppercase">Scores</h5>
        <pre class="mt-2 overflow-auto bg-black p-3 font-mono text-sm font-bold text-lime-300">{{ formatJson(state.data.scores) }}</pre>
      </div>

      <div v-if="state.data.error" class="border-4 border-black bg-red-50 p-3">
        <h5 class="font-mono text-xs font-black uppercase text-red-800">Stored error</h5>
        <pre class="mt-2 overflow-auto font-mono text-sm font-bold text-red-800">{{ formatJson(state.data.error) }}</pre>
      </div>

      <a
        v-if="state.data.mr_url"
        class="inline-flex min-h-11 cursor-pointer items-center justify-center border-4 border-black bg-white px-4 py-2 font-black uppercase shadow-[4px_4px_0_#000] hover:bg-yellow-200 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-black"
        :href="state.data.mr_url"
        target="_blank"
        rel="noreferrer"
      >
        Open merge request
      </a>
    </article>
  </section>
</template>
