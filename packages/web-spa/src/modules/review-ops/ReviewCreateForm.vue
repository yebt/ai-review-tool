<script setup lang="ts">
import { computed, reactive } from 'vue'
import { Play } from '@lucide/vue'

import type { AsyncState, CreateReviewRequest, ReviewFormDraft, ReviewRecord } from './types'

const props = defineProps<{
  state: Readonly<AsyncState<ReviewRecord>>
}>()

const emit = defineEmits<{
  submitReview: [request: CreateReviewRequest]
}>()

const draft = reactive<ReviewFormDraft>({
  projectUrl: 'https://gitlab.com/acme/widget',
  projectPath: '',
  mrIid: '7',
})

const isSubmitting = computed(() => props.state.status === 'loading')
const canSubmit = computed(() => {
  return Number.parseInt(draft.mrIid, 10) > 0 && (draft.projectUrl.trim() !== '' || draft.projectPath.trim() !== '')
})

function submit() {
  if (!canSubmit.value || isSubmitting.value) {
    return
  }

  emit('submitReview', {
    platform: 'gitlab',
    project_url: draft.projectUrl.trim(),
    project_path: draft.projectPath.trim(),
    mr_iid: Number.parseInt(draft.mrIid, 10),
  })
}
</script>

<template>
  <form class="border-4 border-black bg-zinc-50 p-4" @submit.prevent="submit">
    <div>
      <p class="font-mono text-xs font-black uppercase tracking-[0.2em]">POST /api/v1/reviews</p>
      <h3 class="mt-1 text-2xl font-black uppercase">Create review</h3>
    </div>

    <div class="mt-4 space-y-4">
      <label class="block font-bold" for="review-project-url">
        Project URL
        <input
          id="review-project-url"
          v-model="draft.projectUrl"
          type="url"
          class="mt-1 block min-h-11 w-full border-4 border-black bg-white px-3 py-2 font-mono text-sm focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-black"
          placeholder="https://gitlab.com/group/project"
        />
      </label>

      <label class="block font-bold" for="review-project-path">
        Project path override
        <input
          id="review-project-path"
          v-model="draft.projectPath"
          type="text"
          class="mt-1 block min-h-11 w-full border-4 border-black bg-white px-3 py-2 font-mono text-sm focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-black"
          placeholder="group/project"
        />
      </label>

      <label class="block font-bold" for="review-mr-iid">
        Merge request IID
        <input
          id="review-mr-iid"
          v-model="draft.mrIid"
          type="number"
          min="1"
          class="mt-1 block min-h-11 w-full border-4 border-black bg-white px-3 py-2 font-mono text-sm focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-black"
        />
      </label>
    </div>

    <button
      type="submit"
      class="mt-5 inline-flex min-h-11 w-full cursor-pointer items-center justify-center gap-2 border-4 border-black bg-yellow-300 px-4 py-3 font-black uppercase shadow-[4px_4px_0_#000] transition-colors hover:bg-lime-300 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-black disabled:cursor-not-allowed disabled:opacity-60"
      :disabled="!canSubmit || isSubmitting"
    >
      <Play class="size-5" aria-hidden="true" />
      {{ isSubmitting ? 'Running review' : 'Create review' }}
    </button>

    <p v-if="state.status === 'error'" class="mt-4 border-4 border-black bg-red-50 p-3 font-mono text-sm font-bold text-red-800" role="alert">
      {{ state.error }}
    </p>
    <p v-else-if="state.status === 'success'" class="mt-4 border-4 border-black bg-lime-200 p-3 font-bold">
      Created review {{ state.data?.id }} with status {{ state.data?.status }}.
    </p>
  </form>
</template>
