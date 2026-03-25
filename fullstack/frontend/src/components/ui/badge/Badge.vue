<script setup>
import { computed } from 'vue'

import { cn } from '@/lib/utils'

const props = defineProps({
  variant: {
    type: String,
    default: 'default',
  },
  class: {
    type: String,
    default: '',
  },
})

const classes = computed(() =>
  cn(
    'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors',
    {
      'border-transparent bg-primary/10 text-primary': props.variant === 'default',
      'border-transparent bg-success/15 text-success': props.variant === 'success',
      'border-transparent bg-warning/15 text-warning': props.variant === 'warning',
      'border-transparent bg-danger/15 text-danger': props.variant === 'danger',
      'border-border bg-muted text-muted-foreground': props.variant === 'outline',
    },
    props.class,
  ),
)
</script>

<template>
  <span :class="classes"><slot /></span>
</template>
