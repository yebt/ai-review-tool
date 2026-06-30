<script setup lang="ts">
const availableActions = [
  {
    title: 'Backend health',
    route: '/health',
    endpoint: 'GET /healthz',
    description: 'Check the Go server liveness endpoint without loading other API data.',
  },
  {
    title: 'Loaded 4R skills',
    route: '/skills',
    endpoint: 'GET /api/v1/skills',
    description: 'Inspect loaded review skill metadata only; prompt bodies and file paths stay hidden.',
  },
  {
    title: 'Review operations',
    route: '/reviews',
    endpoint: 'POST/GET /api/v1/reviews',
    description: 'Create manual GitLab MR reviews, list history, inspect comments, and watch SSE events.',
  },
]

const upcomingActions = [
  'Repo CRUD and repo memory management',
  'Publish and approval workflows',
  'Final Phase 7 review UI',
  'Go CLI workflow',
]
</script>

<template>
  <section class="space-y-8">
    <header class="border-4 border-black bg-white p-5 shadow-[8px_8px_0_#000]">
      <p class="font-mono text-sm font-bold uppercase tracking-[0.24em]">Phase 4.5 navigation layer</p>
      <h1 class="mt-2 text-4xl font-black uppercase tracking-[-0.04em] sm:text-6xl">
        Capability map
      </h1>
      <p class="mt-3 max-w-4xl font-bold text-zinc-700">
        This temporary brutalist SPA separates each available action into its own route. Home is only a checkpoint map; it does not load backend health, skills, or review operation data.
      </p>
    </header>

    <section aria-labelledby="available-actions-heading">
      <h2 id="available-actions-heading" class="text-2xl font-black uppercase">Available now</h2>
      <div class="mt-4 grid gap-5 lg:grid-cols-3">
        <RouterLink
          v-for="action in availableActions"
          :key="action.route"
          :to="action.route"
          class="block min-h-44 cursor-pointer border-4 border-black bg-white p-5 shadow-[6px_6px_0_#000] transition-colors hover:bg-yellow-100 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-black"
        >
          <p class="font-mono text-xs font-black uppercase tracking-[0.18em]">{{ action.endpoint }}</p>
          <h3 class="mt-3 text-2xl font-black uppercase">{{ action.title }}</h3>
          <p class="mt-3 font-bold text-zinc-700">{{ action.description }}</p>
        </RouterLink>
      </div>
    </section>

    <section class="grid gap-5 lg:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)]" aria-labelledby="upcoming-actions-heading">
      <div class="border-4 border-black bg-yellow-100 p-5">
        <h2 id="upcoming-actions-heading" class="text-2xl font-black uppercase">Not available yet</h2>
        <ul class="mt-4 list-square space-y-2 pl-5 font-bold">
          <li v-for="action in upcomingActions" :key="action">{{ action }}</li>
        </ul>
      </div>

      <RouterLink
        to="/repos"
        class="block cursor-pointer border-4 border-black bg-white p-5 shadow-[6px_6px_0_#000] transition-colors hover:bg-zinc-200 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-black"
      >
        <p class="font-mono text-xs font-black uppercase tracking-[0.18em]">Phase 5 placeholder</p>
        <h3 class="mt-3 text-2xl font-black uppercase">Repo CRUD status</h3>
        <p class="mt-3 font-bold text-zinc-700">
          Opens the repo placeholder page so the missing Phase 5 boundary is visible without adding backend behavior early.
        </p>
      </RouterLink>
    </section>
  </section>
</template>
