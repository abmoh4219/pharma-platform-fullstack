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
  <div :class="cn('rounded-xl border border-border bg-card text-card-foreground shadow-card', $props.class)">
    <slot />
  </div>
</template>
