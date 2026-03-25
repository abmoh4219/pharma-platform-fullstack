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
  <div :class="cn('p-5 pt-3', $props.class)">
    <slot />
  </div>
</template>
