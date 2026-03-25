<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ShieldCheck } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

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
const activeTab = ref('qualifications')

const qualifications = ref([])
const restrictions = ref([])

const deleting = ref(false)
const deleteTarget = reactive({
  type: '',
  id: null,
})
const deleteDialogOpen = ref(false)

const qualificationForm = reactive({
  id: null,
  entity_type: 'supplier',
  entity_name: '',
  qualification_code: '',
  issue_date: '',
  expiry_date: '',
  status: 'active',
  notes: '',
})

const restrictionForm = reactive({
  id: null,
  med_name: '',
  rule_type: 'controlled_purchase_limit',
  max_quantity: 1,
  requires_approval: true,
  is_active: true,
})

const checkForm = reactive({
  med_name: '',
  quantity: '',
})
const checkResult = ref(null)

const qualificationRows = computed(() => qualifications.value)
const restrictionRows = computed(() => restrictions.value)

function expiryVariant(days) {
  if (days < 0) return 'danger'
  if (days <= 30) return 'warning'
  return 'success'
}

async function loadData() {
  loading.value = true
  try {
    const [qualData, restData] = await Promise.all([api.listQualifications(), api.listRestrictions()])
    qualifications.value = qualData
    restrictions.value = restData
  } catch (err) {
    toast.error(err.message || 'Failed to load compliance data')
  } finally {
    loading.value = false
  }
}

function resetQualification() {
  qualificationForm.id = null
  qualificationForm.entity_type = 'supplier'
  qualificationForm.entity_name = ''
  qualificationForm.qualification_code = ''
  qualificationForm.issue_date = ''
  qualificationForm.expiry_date = ''
  qualificationForm.status = 'active'
  qualificationForm.notes = ''
}

function editQualification(row) {
  qualificationForm.id = row.id
  qualificationForm.entity_type = row.entity_type
  qualificationForm.entity_name = row.entity_name
  qualificationForm.qualification_code = row.qualification_code
  qualificationForm.issue_date = row.issue_date
  qualificationForm.expiry_date = row.expiry_date
  qualificationForm.status = row.status
  qualificationForm.notes = ''
}

async function saveQualification() {
  if (
    !qualificationForm.entity_name.trim() ||
    !qualificationForm.qualification_code.trim() ||
    !qualificationForm.issue_date ||
    !qualificationForm.expiry_date
  ) {
    toast.warning('Please complete required qualification fields')
    return
  }

  try {
    const payload = { ...qualificationForm }
    if (qualificationForm.id) {
      await api.updateQualification(qualificationForm.id, payload)
      toast.success('Qualification updated')
    } else {
      await api.createQualification(payload)
      toast.success('Qualification created')
    }
    resetQualification()
    await loadData()
  } catch (err) {
    toast.error(err.message || 'Failed to save qualification')
  }
}

function requestDeleteQualification(id) {
  deleteTarget.type = 'qualification'
  deleteTarget.id = id
  deleteDialogOpen.value = true
}

function resetRestriction() {
  restrictionForm.id = null
  restrictionForm.med_name = ''
  restrictionForm.rule_type = 'controlled_purchase_limit'
  restrictionForm.max_quantity = 1
  restrictionForm.requires_approval = true
  restrictionForm.is_active = true
}

function editRestriction(row) {
  restrictionForm.id = row.id
  restrictionForm.med_name = row.med_name
  restrictionForm.rule_type = row.rule_type
  restrictionForm.max_quantity = row.max_quantity
  restrictionForm.requires_approval = row.requires_approval
  restrictionForm.is_active = row.is_active
}

async function saveRestriction() {
  if (!restrictionForm.med_name.trim() || !restrictionForm.rule_type.trim() || Number(restrictionForm.max_quantity) <= 0) {
    toast.warning('Please complete required restriction fields')
    return
  }

  try {
    const payload = {
      ...restrictionForm,
      max_quantity: Number(restrictionForm.max_quantity),
    }
    if (restrictionForm.id) {
      await api.updateRestriction(restrictionForm.id, payload)
      toast.success('Restriction updated')
    } else {
      await api.createRestriction(payload)
      toast.success('Restriction created')
    }
    resetRestriction()
    await loadData()
  } catch (err) {
    toast.error(err.message || 'Failed to save restriction')
  }
}

function requestDeleteRestriction(id) {
  deleteTarget.type = 'restriction'
  deleteTarget.id = id
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (deleting.value) return
  if (!deleteTarget.id || !deleteTarget.type) return

  deleting.value = true
  try {
    if (deleteTarget.type === 'qualification') {
      await api.deleteQualification(deleteTarget.id)
      toast.success('Qualification deleted')
    } else {
      await api.deleteRestriction(deleteTarget.id)
      toast.success('Restriction deleted')
    }
    deleteDialogOpen.value = false
    deleteTarget.type = ''
    deleteTarget.id = null
    await loadData()
  } catch (err) {
    toast.error(err.message || 'Delete failed')
  } finally {
    deleting.value = false
  }
}

async function checkControlledRule() {
  if (!checkForm.med_name.trim() || Number(checkForm.quantity) <= 0) {
    toast.warning('Enter medication and positive quantity')
    return
  }

  try {
    checkResult.value = await api.checkRestriction({
      med_name: checkForm.med_name,
      quantity: Number(checkForm.quantity),
    })
  } catch (err) {
    toast.error(err.message || 'Failed to check restriction')
  }
}

onMounted(loadData)
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div>
        <h2 class="section-title">Compliance Console</h2>
        <p class="section-subtitle">Qualification lifecycle and controlled medication policy management</p>
      </div>
      <Badge variant="outline">{{ qualificationRows.length }} qualification(s)</Badge>
    </div>

    <div class="rounded-xl border border-border bg-card p-2">
      <div class="flex flex-wrap gap-2">
        <Button
          :variant="activeTab === 'qualifications' ? 'default' : 'outline'"
          size="sm"
          @click="activeTab = 'qualifications'"
        >
          Qualifications
        </Button>
        <Button
          :variant="activeTab === 'restrictions' ? 'default' : 'outline'"
          size="sm"
          @click="activeTab = 'restrictions'"
        >
          Controlled Meds Rules
        </Button>
      </div>
    </div>

    <div v-if="loading" class="rounded-xl border border-border bg-card p-4">
      <Spinner>Loading compliance workspace...</Spinner>
    </div>

    <template v-else>
      <template v-if="activeTab === 'qualifications'">
        <div class="panel-grid-2">
          <Card>
            <CardHeader class="pb-2">
              <CardTitle>{{ qualificationForm.id ? 'Edit Qualification' : 'Create Qualification' }}</CardTitle>
            </CardHeader>
            <CardContent class="space-y-3">
              <div class="grid grid-cols-1 gap-3 lg:grid-cols-2">
                <div>
                  <label class="field-label">Entity type</label>
                  <select v-model="qualificationForm.entity_type" class="form-select">
                    <option value="client">client</option>
                    <option value="supplier">supplier</option>
                  </select>
                </div>
                <div>
                  <label class="field-label">Entity name</label>
                  <Input v-model="qualificationForm.entity_name" placeholder="Entity legal name" />
                </div>
                <div>
                  <label class="field-label">Qualification code</label>
                  <Input v-model="qualificationForm.qualification_code" placeholder="QF-2026-001" />
                </div>
                <div>
                  <label class="field-label">Status</label>
                  <select v-model="qualificationForm.status" class="form-select">
                    <option value="active">active</option>
                    <option value="inactive">inactive</option>
                  </select>
                </div>
                <div>
                  <label class="field-label">Issue date</label>
                  <Input v-model="qualificationForm.issue_date" type="date" />
                </div>
                <div>
                  <label class="field-label">Expiry date</label>
                  <Input v-model="qualificationForm.expiry_date" type="date" />
                </div>
              </div>

              <div>
                <label class="field-label">Notes (encrypted)</label>
                <Textarea v-model="qualificationForm.notes" :rows="3" placeholder="Confidential notes" />
              </div>

              <div class="flex flex-wrap gap-2">
                <Button @click="saveQualification">{{ qualificationForm.id ? 'Update' : 'Create' }} Qualification</Button>
                <Button variant="outline" @click="resetQualification">Reset</Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader class="pb-2">
              <CardTitle>Expiration Guidance</CardTitle>
            </CardHeader>
            <CardContent class="space-y-3 text-sm">
              <div class="rounded-lg border border-danger/30 bg-danger/10 p-3 text-danger">
                Red highlight appears for records expiring within 30 days.
              </div>
              <div class="rounded-lg border border-warning/30 bg-warning/10 p-3 text-warning">
                Expired records are auto-deactivated by backend policy.
              </div>
              <div class="rounded-lg border border-success/30 bg-success/10 p-3 text-success">
                Keep issue and expiry dates accurate to prevent procurement delays.
              </div>
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader class="pb-2">
            <CardTitle>Qualification Register</CardTitle>
          </CardHeader>
          <CardContent class="overflow-x-auto p-0">
            <table class="min-w-full divide-y divide-border text-sm">
              <thead class="bg-muted/60 text-left text-xs uppercase tracking-wide text-muted-foreground">
                <tr>
                  <th class="px-4 py-3">ID</th>
                  <th class="px-4 py-3">Type</th>
                  <th class="px-4 py-3">Entity</th>
                  <th class="px-4 py-3">Code</th>
                  <th class="px-4 py-3">Expiry</th>
                  <th class="px-4 py-3">Countdown</th>
                  <th class="px-4 py-3">Status</th>
                  <th class="px-4 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border">
                <tr
                  v-for="row in qualificationRows"
                  :key="row.id"
                  :class="row.highlight_red ? 'bg-danger/5' : 'hover:bg-accent/40'"
                >
                  <td class="px-4 py-3">{{ row.id }}</td>
                  <td class="px-4 py-3 capitalize">{{ row.entity_type }}</td>
                  <td class="px-4 py-3 font-medium text-foreground">{{ row.entity_name }}</td>
                  <td class="px-4 py-3">{{ row.qualification_code }}</td>
                  <td class="px-4 py-3">{{ row.expiry_date }}</td>
                  <td class="px-4 py-3">
                    <Badge :variant="expiryVariant(row.days_to_expiry)">
                      {{ row.days_to_expiry }} day(s)
                    </Badge>
                  </td>
                  <td class="px-4 py-3">
                    <Badge :variant="row.status === 'active' ? 'success' : 'outline'">{{ row.status }}</Badge>
                  </td>
                  <td class="px-4 py-3 text-right">
                    <div class="inline-flex gap-2">
                      <Button size="sm" variant="ghost" @click="editQualification(row)">Edit</Button>
                      <Button size="sm" variant="danger" @click="requestDeleteQualification(row.id)">Delete</Button>
                    </div>
                  </td>
                </tr>
                <tr v-if="!qualificationRows.length">
                  <td colspan="8" class="px-4 py-6 text-center text-muted-foreground">No qualifications found.</td>
                </tr>
              </tbody>
            </table>
          </CardContent>
        </Card>
      </template>

      <template v-else>
        <div class="panel-grid-2">
          <Card>
            <CardHeader class="pb-2">
              <CardTitle>{{ restrictionForm.id ? 'Edit Restriction' : 'Create Restriction' }}</CardTitle>
            </CardHeader>
            <CardContent class="space-y-3">
              <div class="grid grid-cols-1 gap-3 lg:grid-cols-2">
                <div>
                  <label class="field-label">Medication</label>
                  <Input v-model="restrictionForm.med_name" placeholder="Morphine" />
                </div>
                <div>
                  <label class="field-label">Rule type</label>
                  <Input v-model="restrictionForm.rule_type" placeholder="controlled_purchase_limit" />
                </div>
                <div>
                  <label class="field-label">Max quantity</label>
                  <Input v-model="restrictionForm.max_quantity" type="number" min="1" step="0.01" />
                </div>
                <div>
                  <label class="field-label">Requires approval</label>
                  <select v-model="restrictionForm.requires_approval" class="form-select">
                    <option :value="true">Yes</option>
                    <option :value="false">No</option>
                  </select>
                </div>
                <div>
                  <label class="field-label">Active</label>
                  <select v-model="restrictionForm.is_active" class="form-select">
                    <option :value="true">Active</option>
                    <option :value="false">Inactive</option>
                  </select>
                </div>
              </div>

              <div class="flex flex-wrap gap-2">
                <Button @click="saveRestriction">{{ restrictionForm.id ? 'Update' : 'Create' }} Restriction</Button>
                <Button variant="outline" @click="resetRestriction">Reset</Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader class="pb-2">
              <CardTitle>Purchase Rule Checker</CardTitle>
            </CardHeader>
            <CardContent class="space-y-3">
              <div>
                <label class="field-label">Medication</label>
                <Input v-model="checkForm.med_name" placeholder="Morphine" />
              </div>
              <div>
                <label class="field-label">Quantity</label>
                <Input v-model="checkForm.quantity" type="number" min="1" step="0.01" />
              </div>
              <Button variant="outline" @click="checkControlledRule">
                <ShieldCheck class="h-4 w-4" />
                Check Rule
              </Button>

              <div v-if="checkResult" class="rounded-lg border p-3" :class="checkResult.allowed ? 'border-success/40 bg-success/10 text-success' : 'border-danger/40 bg-danger/10 text-danger'">
                <p class="font-semibold">{{ checkResult.allowed ? 'Purchase Allowed' : 'Purchase Blocked' }}</p>
                <p class="text-xs">Reason: {{ checkResult.reason }}</p>
              </div>
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader class="pb-2">
            <CardTitle>Controlled Medication Restrictions</CardTitle>
          </CardHeader>
          <CardContent class="overflow-x-auto p-0">
            <table class="min-w-full divide-y divide-border text-sm">
              <thead class="bg-muted/60 text-left text-xs uppercase tracking-wide text-muted-foreground">
                <tr>
                  <th class="px-4 py-3">ID</th>
                  <th class="px-4 py-3">Medication</th>
                  <th class="px-4 py-3">Rule</th>
                  <th class="px-4 py-3">Max Qty</th>
                  <th class="px-4 py-3">Approval</th>
                  <th class="px-4 py-3">State</th>
                  <th class="px-4 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border">
                <tr v-for="row in restrictionRows" :key="row.id" class="hover:bg-accent/40">
                  <td class="px-4 py-3">{{ row.id }}</td>
                  <td class="px-4 py-3 font-medium text-foreground">{{ row.med_name }}</td>
                  <td class="px-4 py-3">{{ row.rule_type }}</td>
                  <td class="px-4 py-3">{{ row.max_quantity }}</td>
                  <td class="px-4 py-3">
                    <Badge :variant="row.requires_approval ? 'warning' : 'success'">
                      {{ row.requires_approval ? 'Required' : 'Not required' }}
                    </Badge>
                  </td>
                  <td class="px-4 py-3">
                    <Badge :variant="row.is_active ? 'success' : 'outline'">{{ row.is_active ? 'Active' : 'Inactive' }}</Badge>
                  </td>
                  <td class="px-4 py-3 text-right">
                    <div class="inline-flex gap-2">
                      <Button size="sm" variant="ghost" @click="editRestriction(row)">Edit</Button>
                      <Button size="sm" variant="danger" @click="requestDeleteRestriction(row.id)">Delete</Button>
                    </div>
                  </td>
                </tr>
                <tr v-if="!restrictionRows.length">
                  <td colspan="7" class="px-4 py-6 text-center text-muted-foreground">No restriction rules found.</td>
                </tr>
              </tbody>
            </table>
          </CardContent>
        </Card>
      </template>
    </template>

    <ConfirmDialog
      v-model:open="deleteDialogOpen"
      title="Delete compliance record"
      description="This deletion is permanent and will be captured in audit logs."
      confirm-text="Delete"
      cancel-text="Cancel"
      :loading="deleting"
      @confirm="confirmDelete"
    />
  </div>
</template>
