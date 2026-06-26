<script setup lang="ts">
import { computed } from 'vue'

import type { CheckStatus } from './types'

const props = defineProps<{
  status: CheckStatus
}>()

const label = computed(() => {
  switch (props.status) {
    case 'idle':
      return 'Not loaded'
    case 'loading':
      return 'Loading'
    case 'success':
      return 'Success'
    case 'error':
      return 'Error'
    default:
      return 'Unknown'
  }
})

const badgeClass = computed(() => ({
  'border-black bg-white text-black': props.status === 'idle',
  'border-black bg-yellow-300 text-black': props.status === 'loading',
  'border-black bg-lime-300 text-black': props.status === 'success',
  'border-black bg-red-800 text-white': props.status === 'error',
}))
</script>

<template>
  <span
    class="inline-flex border-2 px-3 py-1 text-xs font-black uppercase tracking-[0.2em]"
    :class="badgeClass"
  >
    {{ label }}
  </span>
</template>
