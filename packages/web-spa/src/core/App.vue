<script setup lang="ts">
const navigationItems = [
  { to: '/', label: 'Home' },
  { to: '/health', label: 'Health' },
  { to: '/skills', label: 'Skills' },
  { to: '/reviews', label: 'Reviews' },
  { to: '/repos', label: 'Repos' },
]
</script>

<template>
  <div class="min-h-screen bg-zinc-100 text-black">
    <header class="border-b-4 border-black bg-white">
      <div class="mx-auto flex max-w-7xl flex-col gap-4 px-4 py-4 sm:px-6 lg:flex-row lg:items-center lg:justify-between lg:px-8">
        <RouterLink
          to="/"
          class="w-fit font-mono text-sm font-black uppercase tracking-[0.24em] focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-black"
        >
          Co-Review Smoke UI
        </RouterLink>
        <nav aria-label="Primary navigation">
          <ul class="flex flex-wrap gap-2">
            <li v-for="item in navigationItems" :key="item.to">
              <RouterLink
                :to="item.to"
                class="inline-flex min-h-11 cursor-pointer items-center border-2 border-black bg-white px-3 py-2 font-black uppercase transition-colors hover:bg-yellow-300 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-black"
                active-class="bg-lime-300"
              >
                {{ item.label }}
              </RouterLink>
            </li>
          </ul>
        </nav>
      </div>
    </header>

    <main class="px-4 py-6 sm:px-6 lg:px-8">
      <div class="mx-auto max-w-7xl">
        <RouterView />
      </div>
    </main>
  </div>
</template>
