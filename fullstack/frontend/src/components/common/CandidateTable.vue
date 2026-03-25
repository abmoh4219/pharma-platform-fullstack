<script setup>
import { Search, SquarePen } from 'lucide-vue-next'

import Badge from '@/components/ui/badge/Badge.vue'
import Button from '@/components/ui/button/Button.vue'

const props = defineProps({
  candidates: {
    type: Array,
    default: () => [],
  },
  loading: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['edit', 'open-search'])
</script>

<template>
  <div class="rounded-lg border border-border bg-card shadow-card">
    <div class="flex items-center justify-between border-b border-border p-4">
      <h3 class="text-sm font-semibold text-foreground">Candidates</h3>
      <Button variant="outline" size="sm" @click="emit('open-search')">
        <Search class="h-4 w-4" />
        Search
      </Button>
    </div>

    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-border text-sm">
        <thead class="bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
          <tr>
            <th class="px-4 py-3">ID</th>
            <th class="px-4 py-3">Name</th>
            <th class="px-4 py-3">Phone</th>
            <th class="px-4 py-3">ID Number</th>
            <th class="px-4 py-3">Position</th>
            <th class="px-4 py-3">Status</th>
            <th class="px-4 py-3 text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-if="loading">
            <td colspan="7" class="px-4 py-5 text-center text-muted-foreground">Loading candidates...</td>
          </tr>
          <tr v-else-if="!candidates.length">
            <td colspan="7" class="px-4 py-5 text-center text-muted-foreground">No candidates found.</td>
          </tr>
          <tr
            v-for="candidate in candidates"
            :key="candidate.id"
            class="transition-colors hover:bg-accent/50"
            data-testid="candidate-row"
          >
            <td class="px-4 py-3">{{ candidate.id }}</td>
            <td class="px-4 py-3 font-medium text-foreground">{{ candidate.full_name }}</td>
            <td class="px-4 py-3">{{ candidate.phone }}</td>
            <td class="px-4 py-3">{{ candidate.id_number }}</td>
            <td class="px-4 py-3">{{ candidate.position_title || '-' }}</td>
            <td class="px-4 py-3">
              <Badge variant="outline" class="capitalize">{{ candidate.status }}</Badge>
            </td>
            <td class="px-4 py-3 text-right">
              <Button variant="ghost" size="sm" @click="emit('edit', candidate)">
                <SquarePen class="h-4 w-4" />
                Edit
              </Button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
