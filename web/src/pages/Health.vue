<script setup lang="ts">
import { ref } from 'vue'
import { getHealth } from '../services/api'
import { useAsync } from '../composables/useAsync'
import Card from '../components/ui/Card.vue'
import Button from '../components/ui/Button.vue'

const result = ref<any>(null)
const { loading, run } = useAsync(getHealth)

async function check() {
  result.value = await run()
}
</script>

<template>
  <Card>
    <template #header>
      <h2 class="text-lg font-semibold">健康检查</h2>
    </template>
    <div class="space-y-4">
      <Button @click="check" :disabled="loading">{{ loading ? '检查中...' : '检查' }}</Button>
      <pre v-if="result" class="rounded-md border bg-gray-50 p-3 text-sm overflow-auto">{{ result }}</pre>
    </div>
  </Card>
</template>