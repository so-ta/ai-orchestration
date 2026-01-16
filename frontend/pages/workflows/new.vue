<script setup lang="ts">
const projects = useProjects()

const form = ref({
  name: '',
  description: '',
})
const loading = ref(false)
const error = ref<string | null>(null)

async function handleSubmit() {
  if (!form.value.name.trim()) {
    error.value = 'Name is required'
    return
  }

  try {
    loading.value = true
    error.value = null
    const response = await projects.create({
      name: form.value.name,
      description: form.value.description,
    })
    navigateTo(`/workflows/${response.data.id}`)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to create workflow'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <h1 style="font-size: 1.5rem; font-weight: 600; margin-bottom: 1.5rem;">
      New Workflow
    </h1>

    <div class="card" style="max-width: 600px;">
      <form @submit.prevent="handleSubmit">
        <div v-if="error" style="color: var(--color-error); margin-bottom: 1rem;">
          {{ error }}
        </div>

        <div class="form-group">
          <label class="form-label">Name *</label>
          <input
            v-model="form.name"
            type="text"
            class="form-input"
            placeholder="My Workflow"
            required
          >
        </div>

        <div class="form-group">
          <label class="form-label">Description</label>
          <textarea
            v-model="form.description"
            class="form-input"
            rows="3"
            placeholder="Describe what this workflow does..."
          />
        </div>

        <div class="flex gap-2">
          <button type="submit" class="btn btn-primary" :disabled="loading">
            {{ loading ? 'Creating...' : 'Create Workflow' }}
          </button>
          <NuxtLink to="/workflows" class="btn btn-outline">
            Cancel
          </NuxtLink>
        </div>
      </form>
    </div>
  </div>
</template>
