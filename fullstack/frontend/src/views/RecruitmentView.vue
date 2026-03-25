<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { CopyPlus, Search, UploadCloud } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

import CandidateTable from '@/components/common/CandidateTable.vue'
import MatchScoreBadge from '@/components/common/MatchScoreBadge.vue'
import Badge from '@/components/ui/badge/Badge.vue'
import Button from '@/components/ui/button/Button.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import CardHeader from '@/components/ui/card/CardHeader.vue'
import CardTitle from '@/components/ui/card/CardTitle.vue'
import ConfirmDialog from '@/components/ui/dialog/ConfirmDialog.vue'
import Input from '@/components/ui/input/Input.vue'
import Spinner from '@/components/ui/spinner/Spinner.vue'
import Textarea from '@/components/ui/textarea/Textarea.vue'

import { api } from '../services/api'

const loading = ref(false)
const importLoading = ref(false)
const savingPosition = ref(false)
const savingCandidate = ref(false)
const searchLoading = ref(false)
const mergeLoading = ref(false)
const mergeDialogOpen = ref(false)

const fileInput = ref(null)

const positions = ref([])
const candidates = ref([])
const searchResults = ref([])

const candidateFilters = reactive({
  q: '',
  status: '',
})

const positionForm = reactive({
  title: '',
  description: '',
})

const candidateForm = reactive({
  id: null,
  full_name: '',
  phone: '',
  id_number: '',
  email: '',
  position_id: '',
  status: 'new',
})

const mergeForm = reactive({
  primary_candidate_id: '',
  duplicate_ids: '',
})

const searchQuery = ref('')

const filteredCandidates = computed(() => {
  const query = candidateFilters.q.trim().toLowerCase()
  return candidates.value.filter((candidate) => {
    if (candidateFilters.status && candidate.status !== candidateFilters.status) {
      return false
    }
    if (!query) return true

    return [candidate.full_name, candidate.phone, candidate.id_number, candidate.position_title]
      .join(' ')
      .toLowerCase()
      .includes(query)
  })
})

const spotlightCandidates = computed(() => filteredCandidates.value.slice(0, 4))

function statusVariant(status) {
  if (status === 'shortlisted') return 'success'
  if (status === 'rejected') return 'danger'
  if (status === 'imported') return 'warning'
  return 'outline'
}

async function loadData() {
  loading.value = true
  try {
    const [pos, cands] = await Promise.all([api.listPositions(), api.listCandidates()])
    positions.value = pos
    candidates.value = cands
  } catch (err) {
    toast.error(err.message || 'Failed to load recruitment data')
  } finally {
    loading.value = false
  }
}

async function createPosition() {
  if (savingPosition.value) return
  if (!positionForm.title.trim()) {
    toast.warning('Position title is required')
    return
  }

  savingPosition.value = true
  try {
    await api.createPosition({
      title: positionForm.title,
      description: positionForm.description,
    })
    toast.success('Position created')
    positionForm.title = ''
    positionForm.description = ''
    await loadData()
  } catch (err) {
    toast.error(err.message || 'Failed to create position')
  } finally {
    savingPosition.value = false
  }
}

function resetCandidateForm() {
  candidateForm.id = null
  candidateForm.full_name = ''
  candidateForm.phone = ''
  candidateForm.id_number = ''
  candidateForm.email = ''
  candidateForm.position_id = ''
  candidateForm.status = 'new'
}

function editCandidate(row) {
  candidateForm.id = row.id
  candidateForm.full_name = row.full_name
  candidateForm.phone = ''
  candidateForm.id_number = ''
  candidateForm.email = row.email
  candidateForm.position_id = row.position_id || ''
  candidateForm.status = row.status || 'new'
  toast.info('Provide a new phone and ID number to save masked fields.')
}

async function submitCandidate() {
  if (savingCandidate.value) return
  if (!candidateForm.full_name.trim() || !candidateForm.phone.trim() || !candidateForm.id_number.trim()) {
    toast.warning('Full name, phone, and ID number are required')
    return
  }

  const payload = {
    full_name: candidateForm.full_name,
    phone: candidateForm.phone,
    id_number: candidateForm.id_number,
    email: candidateForm.email,
    position_id: candidateForm.position_id ? Number(candidateForm.position_id) : null,
    status: candidateForm.status,
  }

  savingCandidate.value = true
  try {
    if (candidateForm.id) {
      await api.updateCandidate(candidateForm.id, payload)
      toast.success('Candidate updated')
    } else {
      await api.createCandidate(payload)
      toast.success('Candidate created')
    }
    resetCandidateForm()
    await loadData()
  } catch (err) {
    toast.error(err.message || 'Failed to save candidate')
  } finally {
    savingCandidate.value = false
  }
}

function triggerImport() {
  fileInput.value?.click()
}

async function onFileSelected(event) {
  const file = event.target.files?.[0]
  if (!file) return

  importLoading.value = true
  try {
    const result = await api.importCandidates(file)
    toast.success(`Imported ${result.imported} candidates`)
    if (Array.isArray(result.failed) && result.failed.length) {
      toast.warning(`${result.failed.length} rows failed validation`)
    }
    await loadData()
  } catch (err) {
    toast.error(err.message || 'Import failed')
  } finally {
    importLoading.value = false
    event.target.value = ''
  }
}

async function runSearch() {
  const query = searchQuery.value.trim()
  if (!query) {
    searchResults.value = []
    return
  }

  searchLoading.value = true
  try {
    searchResults.value = await api.searchCandidates(query)
  } catch (err) {
    toast.error(err.message || 'Search failed')
  } finally {
    searchLoading.value = false
  }
}

function openSearch() {
  const input = document.getElementById('recruitment-search')
  if (input) {
    input.scrollIntoView({ behavior: 'smooth', block: 'center' })
    input.focus()
  }
}

function requestMerge() {
  if (!mergeForm.primary_candidate_id.trim() || !mergeForm.duplicate_ids.trim()) {
    toast.warning('Primary ID and duplicate IDs are required')
    return
  }
  mergeDialogOpen.value = true
}

async function confirmMerge() {
  if (mergeLoading.value) return

  const primary = Number(mergeForm.primary_candidate_id)
  const duplicateIds = mergeForm.duplicate_ids
    .split(',')
    .map((value) => Number(value.trim()))
    .filter((value) => Number.isFinite(value) && value > 0)

  if (!primary || !duplicateIds.length) {
    toast.error('Please enter valid merge IDs')
    return
  }

  mergeLoading.value = true
  try {
    await api.mergeCandidates({
      primary_candidate_id: primary,
      duplicate_ids: duplicateIds,
    })
    toast.success('Duplicates merged successfully')
    mergeForm.primary_candidate_id = ''
    mergeForm.duplicate_ids = ''
    mergeDialogOpen.value = false
    await loadData()
  } catch (err) {
    toast.error(err.message || 'Merge failed')
  } finally {
    mergeLoading.value = false
  }
}

onMounted(loadData)
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div>
        <h2 class="section-title">Recruitment Hub</h2>
        <p class="section-subtitle">Bulk import, deduplicate, and score candidates in one place</p>
      </div>
      <Badge variant="outline">{{ filteredCandidates.length }} candidate(s)</Badge>
    </div>

    <div v-if="loading" class="rounded-xl border border-border bg-card p-4">
      <Spinner>Loading recruitment workspace...</Spinner>
    </div>

    <template v-else>
      <div class="panel-grid-3">
        <Card>
          <CardHeader class="pb-2">
            <CardTitle>Create Position</CardTitle>
          </CardHeader>
          <CardContent class="space-y-3">
            <div>
              <label class="field-label">Title</label>
              <Input v-model="positionForm.title" placeholder="Clinical Pharmacist" />
            </div>
            <div>
              <label class="field-label">Description</label>
              <Textarea v-model="positionForm.description" :rows="3" placeholder="Role responsibilities" />
            </div>
            <Button :loading="savingPosition" @click="createPosition">
              <CopyPlus class="h-4 w-4" />
              Create Position
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader class="pb-2">
            <CardTitle>Bulk Resume Import</CardTitle>
          </CardHeader>
          <CardContent class="space-y-3">
            <p class="text-sm text-muted-foreground">Supported files: CSV/XLSX. Required columns: full_name, phone, id_number, email.</p>
            <input ref="fileInput" type="file" class="hidden" accept=".csv,.xlsx,.xls" @change="onFileSelected" />
            <Button variant="secondary" :loading="importLoading" @click="triggerImport">
              <UploadCloud class="h-4 w-4" />
              Upload Import File
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader class="pb-2">
            <CardTitle>Merge Duplicates</CardTitle>
          </CardHeader>
          <CardContent class="space-y-3">
            <div>
              <label class="field-label">Primary candidate ID</label>
              <Input v-model="mergeForm.primary_candidate_id" placeholder="e.g. 12" />
            </div>
            <div>
              <label class="field-label">Duplicate IDs</label>
              <Input v-model="mergeForm.duplicate_ids" placeholder="e.g. 34, 35" />
            </div>
            <Button variant="outline" @click="requestMerge">Merge Records</Button>
          </CardContent>
        </Card>
      </div>

      <div class="panel-grid-2">
        <Card>
          <CardHeader class="pb-2">
            <CardTitle>{{ candidateForm.id ? 'Edit Candidate' : 'Create Candidate' }}</CardTitle>
          </CardHeader>
          <CardContent class="space-y-3">
            <div class="grid grid-cols-1 gap-3 lg:grid-cols-2">
              <div>
                <label class="field-label">Full name</label>
                <Input v-model="candidateForm.full_name" placeholder="Candidate full name" />
              </div>
              <div>
                <label class="field-label">Email</label>
                <Input v-model="candidateForm.email" placeholder="candidate@example.com" />
              </div>
              <div>
                <label class="field-label">Phone</label>
                <Input v-model="candidateForm.phone" placeholder="raw value required for save" />
              </div>
              <div>
                <label class="field-label">ID number</label>
                <Input v-model="candidateForm.id_number" placeholder="raw value required for save" />
              </div>
              <div>
                <label class="field-label">Position</label>
                <select v-model="candidateForm.position_id" class="form-select">
                  <option value="">Unassigned</option>
                  <option v-for="position in positions" :key="position.id" :value="position.id">{{ position.title }}</option>
                </select>
              </div>
              <div>
                <label class="field-label">Status</label>
                <select v-model="candidateForm.status" class="form-select">
                  <option value="new">new</option>
                  <option value="imported">imported</option>
                  <option value="shortlisted">shortlisted</option>
                  <option value="rejected">rejected</option>
                </select>
              </div>
            </div>
            <div class="flex flex-wrap gap-2">
              <Button :loading="savingCandidate" @click="submitCandidate">
                {{ candidateForm.id ? 'Update Candidate' : 'Create Candidate' }}
              </Button>
              <Button variant="outline" @click="resetCandidateForm">Reset</Button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader class="pb-2">
            <CardTitle>Smart Search</CardTitle>
          </CardHeader>
          <CardContent class="space-y-3">
            <div class="flex gap-2">
              <Input
                id="recruitment-search"
                v-model="searchQuery"
                placeholder="Search by name / phone / ID / email"
                @keyup.enter="runSearch"
              />
              <Button :loading="searchLoading" @click="runSearch">
                <Search class="h-4 w-4" />
                Search
              </Button>
            </div>

            <div v-if="searchResults.length" class="max-h-[280px] space-y-2 overflow-auto pr-1">
              <div
                v-for="result in searchResults"
                :key="result.candidate_id"
                class="rounded-lg border border-border bg-muted/20 p-3"
              >
                <div class="flex items-center justify-between gap-2">
                  <div>
                    <p class="font-medium text-foreground">{{ result.full_name }}</p>
                    <p class="text-xs text-muted-foreground">{{ result.masked_phone }} • {{ result.masked_id }}</p>
                  </div>
                  <MatchScoreBadge :score="result.score" />
                </div>
                <p class="mt-2 text-xs text-muted-foreground">{{ (result.explanation || []).join('; ') || 'No explanation' }}</p>
              </div>
            </div>

            <div v-else class="rounded-lg border border-dashed border-border p-4 text-center text-sm text-muted-foreground">
              No search results yet.
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader class="pb-2">
          <CardTitle>Candidate Spotlight</CardTitle>
        </CardHeader>
        <CardContent>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <div
              v-for="candidate in spotlightCandidates"
              :key="`spotlight-${candidate.id}`"
              class="rounded-lg border border-border bg-background p-3 transition-all hover:shadow-card"
            >
              <div class="flex items-center justify-between gap-2">
                <p class="font-medium text-foreground">{{ candidate.full_name }}</p>
                <Badge :variant="statusVariant(candidate.status)">{{ candidate.status }}</Badge>
              </div>
              <p class="mt-2 text-xs text-muted-foreground">{{ candidate.position_title || 'No position assigned' }}</p>
              <p class="mt-1 text-xs text-muted-foreground">{{ candidate.phone }} • {{ candidate.id_number }}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader class="pb-2">
          <CardTitle>Candidate Directory</CardTitle>
        </CardHeader>
        <CardContent class="space-y-3">
          <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <Input v-model="candidateFilters.q" placeholder="Filter candidates" />
            <select v-model="candidateFilters.status" class="form-select">
              <option value="">All statuses</option>
              <option value="new">new</option>
              <option value="imported">imported</option>
              <option value="shortlisted">shortlisted</option>
              <option value="rejected">rejected</option>
            </select>
          </div>
          <CandidateTable :candidates="filteredCandidates" :loading="false" @edit="editCandidate" @open-search="openSearch" />
        </CardContent>
      </Card>
    </template>

    <ConfirmDialog
      v-model:open="mergeDialogOpen"
      title="Merge duplicate candidates"
      description="Duplicate records will be deleted after merge and their attachments reassigned. Continue?"
      confirm-text="Merge"
      cancel-text="Cancel"
      :loading="mergeLoading"
      @confirm="confirmMerge"
    />
  </div>
</template>
