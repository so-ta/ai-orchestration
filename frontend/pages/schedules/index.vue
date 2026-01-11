<script setup lang="ts">
// Schedules page - placeholder for now
const schedules = ref([])
const loading = ref(false)
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-4">
      <h1 style="font-size: 1.5rem; font-weight: 600;">
        Schedules
      </h1>
      <button class="btn btn-primary" disabled>
        + New Schedule
      </button>
    </div>

    <div class="card" style="padding: 2rem; text-align: center;">
      <p class="text-secondary">
        Schedule management coming soon.
      </p>
      <p class="text-secondary" style="margin-top: 0.5rem;">
        Schedules allow you to run workflows automatically at specified times.
      </p>
    </div>
  </div>
</template>
