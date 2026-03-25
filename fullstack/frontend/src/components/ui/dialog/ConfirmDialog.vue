<script setup>
import Button from '@/components/ui/button/Button.vue'

const open = defineModel('open', {
  type: Boolean,
  default: false,
})

const props = defineProps({
  title: {
    type: String,
    default: 'Confirm action',
  },
  description: {
    type: String,
    default: 'Are you sure you want to continue?',
  },
  confirmText: {
    type: String,
    default: 'Confirm',
  },
  cancelText: {
    type: String,
    default: 'Cancel',
  },
  loading: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['confirm'])

function closeDialog() {
  if (props.loading) return
  open.value = false
}

function onConfirm() {
  emit('confirm')
}
</script>

<template>
  <teleport to="body">
    <div v-if="open" class="fixed inset-0 z-[70] animate-fade-in">
      <button class="absolute inset-0 bg-black/40" type="button" @click="closeDialog" />
      <div class="relative z-[71] mx-auto mt-[18vh] w-[92%] max-w-md rounded-xl border border-border bg-card p-5 shadow-glass">
        <h4 class="text-lg font-semibold text-foreground">{{ title }}</h4>
        <p class="mt-2 text-sm text-muted-foreground">{{ description }}</p>

        <div class="mt-5 flex justify-end gap-2">
          <Button variant="outline" :disabled="loading" @click="closeDialog">{{ cancelText }}</Button>
          <Button variant="danger" :loading="loading" @click="onConfirm">{{ confirmText }}</Button>
        </div>
      </div>
    </div>
  </teleport>
</template>
