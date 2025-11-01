<script setup lang="ts">
defineProps<{ type?: string; modelValue?: string | number; placeholder?: string; disabled?: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', value: string | number): void }>()
</script>

<template>
  <input
    :type="type ?? 'text'"
    :value="modelValue as any"
    :placeholder="placeholder"
    :disabled="disabled"
    @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    class="flex h-9 w-full rounded-md border border-gray-300 bg-white px-3 py-1 text-sm shadow-sm placeholder:text-gray-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 disabled:cursor-not-allowed disabled:opacity-50"
  />
</template>