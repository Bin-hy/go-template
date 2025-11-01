<template>
  <div class="rounded-lg border border-gray-200 bg-white text-gray-900 shadow-sm">
    <div v-if="$slots.header" class="border-b border-gray-200 p-4">
      <slot name="header" />
    </div>
    <div class="p-4">
      <slot />
    </div>
    <div v-if="$slots.footer" class="border-t border-gray-200 p-4">
      <slot name="footer" />
    </div>
  </div>
</template>