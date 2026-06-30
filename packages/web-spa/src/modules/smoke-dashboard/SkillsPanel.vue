<script setup lang="ts">
import { onMounted, shallowRef } from 'vue'
import { ListChecks, RefreshCw } from '@lucide/vue'

import SmokeCard from './SmokeCard.vue'
import type { ReviewSkill } from './types'
import { useSkillsCheck } from './useSmokeChecks'

const { skills, skillsCount, loadSkills } = useSkillsCheck()
const isRefreshing = shallowRef(false)

onMounted(() => {
  void refreshSkills()
})

function getSkillKey(skill: ReviewSkill) {
  return [
    skill.name ?? 'unnamed-skill',
    skill.dimension ?? 'no-dimension',
    skill.harness?.output_schema ?? 'no-schema',
  ].join('::')
}

async function refreshSkills() {
  isRefreshing.value = true

  try {
    await loadSkills()
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
          <p class="font-mono text-sm font-bold uppercase tracking-[0.24em]">GET /api/v1/skills</p>
          <h1 class="mt-2 text-4xl font-black uppercase tracking-[-0.04em] sm:text-6xl">
            Loaded 4R skills
          </h1>
          <p class="mt-3 max-w-3xl font-bold text-zinc-700">
            Metadata-only view of currently loaded review skills. Prompt bodies and internal file paths stay hidden.
          </p>
        </div>
        <button
          type="button"
          class="inline-flex min-h-11 cursor-pointer items-center justify-center gap-2 border-4 border-black bg-yellow-300 px-5 py-3 font-black uppercase shadow-[4px_4px_0_#000] transition-colors hover:bg-lime-300 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-black disabled:cursor-not-allowed disabled:opacity-60"
          :disabled="isRefreshing"
          @click="refreshSkills"
        >
          <RefreshCw class="size-5" :class="{ 'animate-spin': isRefreshing }" aria-hidden="true" />
          Refresh skills
        </button>
      </div>
    </header>

    <SmokeCard title="Review skills" endpoint="GET /api/v1/skills" :status="skills.status">
      <div v-if="skills.status === 'idle'" class="font-bold">Skills have not been requested yet.</div>
      <div v-else-if="skills.status === 'loading'" class="font-bold">Loading review skill metadata...</div>
      <div v-else-if="skills.status === 'error'" class="border-4 border-black bg-red-50 p-4 font-mono text-sm font-bold text-red-800">
        {{ skills.error }}
      </div>
      <div v-else class="space-y-4">
        <div class="flex items-center justify-between border-4 border-black bg-lime-300 p-3">
          <div class="flex items-center gap-3 font-black uppercase">
            <ListChecks class="size-6" aria-hidden="true" />
            Skills loaded
          </div>
          <span class="font-mono text-2xl font-black">{{ skillsCount }}</span>
        </div>

        <div v-if="skillsCount === 0" class="font-bold">The endpoint returned no skills.</div>
        <div v-else class="grid gap-3">
          <article
            v-for="skill in skills.data"
            :key="getSkillKey(skill)"
            class="border-4 border-black bg-zinc-50 p-4"
          >
            <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <h3 class="text-xl font-black uppercase">{{ skill.name ?? 'unnamed-skill' }}</h3>
                <p class="mt-1 text-sm font-bold text-zinc-700">{{ skill.description }}</p>
              </div>
              <span class="w-fit border-2 border-black bg-white px-2 py-1 font-mono text-xs font-black uppercase">
                {{ skill.dimension ?? 'no-dimension' }}
              </span>
            </div>
            <dl class="mt-4 grid gap-2 font-mono text-xs sm:grid-cols-3">
              <div class="border-2 border-black bg-white p-2">
                <dt class="font-black uppercase">Schema</dt>
                <dd>{{ skill.harness?.output_schema ?? 'n/a' }}</dd>
              </div>
              <div class="border-2 border-black bg-white p-2">
                <dt class="font-black uppercase">Retries</dt>
                <dd>{{ skill.harness?.max_retries ?? 'n/a' }}</dd>
              </div>
              <div class="border-2 border-black bg-white p-2">
                <dt class="font-black uppercase">Timeout</dt>
                <dd>{{ skill.harness?.timeout_seconds ?? 'n/a' }}s</dd>
              </div>
            </dl>
          </article>
        </div>
      </div>
    </SmokeCard>
  </section>
</template>
