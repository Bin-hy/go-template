<template>
  <div class="rounded-xl border border-gray-200 bg-white/90 backdrop-blur text-gray-900 shadow-md">
    <div v-if="$slots.header" class="border-b border-gray-200 p-4 bg-gradient-to-r from-gray-50 to-white rounded-t-xl">
      <slot name="header" />
    </div>
    <div class="p-4">
      <slot />
    </div>
    <div v-if="$slots.footer" class="border-t border-gray-200 p-4 bg-gray-50 rounded-b-xl">
      <slot name="footer" />
    </div>
  </div>
</template>