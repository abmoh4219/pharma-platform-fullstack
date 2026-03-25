<script setup>
import { cn } from '@/lib/utils'

const model = defineModel()

defineProps({
  rows: {
    type: Number,
    default: 3,
  },
  placeholder: {
    type: String,
    default: '',
  },
  class: {
    type: String,
    default: '',
  },
})
</script>

<template>
  <textarea
    v-model="model"
    :rows="rows"
    :placeholder="placeholder"
    :class="
      cn(
        'flex min-h-[96px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground',
        'placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        'focus-visible:ring-offset-2 ring-offset-background disabled:cursor-not-allowed disabled:opacity-50',
        $props.class,
      )
    "
  />
</template>
