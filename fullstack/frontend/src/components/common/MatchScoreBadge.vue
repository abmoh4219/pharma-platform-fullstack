<script setup>
import { computed } from 'vue'

import Badge from '@/components/ui/badge/Badge.vue'

const props = defineProps({
  score: {
    type: Number,
    default: 0,
  },
})

const variant = computed(() => {
  if (props.score >= 80) return 'success'
  if (props.score >= 50) return 'warning'
  return 'outline'
})

const label = computed(() => `${Math.max(0, Math.min(100, props.score))}/100`)
</script>

<template>
  <Badge :variant="variant" class="font-bold">
    Match {{ label }}
  </Badge>
</template>
