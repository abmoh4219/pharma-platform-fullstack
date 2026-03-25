<script setup>
import { cn } from '@/lib/utils'

defineProps({
  class: {
    type: String,
    default: '',
  },
})
</script>

<template>
  <h3 :class="cn('text-base font-semibold text-foreground', $props.class)">
    <slot />
  </h3>
</template>
