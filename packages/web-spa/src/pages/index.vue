<script setup lang="ts">
import ReviewOpsDashboard from '@modules/review-ops/ReviewOpsDashboard.vue'
import SmokeDashboard from '@modules/smoke-dashboard/SmokeDashboard.vue'
</script>

<template>
  <main class="min-h-screen bg-zinc-100 px-4 py-6 text-black sm:px-6 lg:px-8">
    <div class="mx-auto max-w-7xl space-y-8">
      <SmokeDashboard />
      <ReviewOpsDashboard />
    </div>
  </main>
</template>
