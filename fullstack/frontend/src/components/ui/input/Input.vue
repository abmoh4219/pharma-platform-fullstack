<script setup>
import { cn } from '@/lib/utils'

const model = defineModel()

defineProps({
  type: {
    type: String,
    default: 'text',
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
  <input
    v-model="model"
    :type="type"
    :placeholder="placeholder"
    :class="
      cn(
        'flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground',
        'placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        'focus-visible:ring-offset-2 ring-offset-background disabled:cursor-not-allowed disabled:opacity-50',
        $props.class,
      )
    "
  />
</template>
