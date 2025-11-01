<script setup lang="ts">
import { cva } from 'class-variance-authority'
import { computed } from 'vue'

type Variant = 'default' | 'secondary' | 'destructive' | 'outline' | 'ghost' | 'link'
type Size = 'sm' | 'md' | 'lg'

import type { Component } from 'vue'
const props = defineProps<{
  variant?: Variant
  size?: Size
  as?: string | Component
  disabled?: boolean
  // 透传路由/链接属性（用于作为 RouterLink 或 a 标签时不触发类型报错）
  to?: unknown
  href?: string
}>()

const buttonVariants = cva(
  'inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50 shadow-sm',
  {
    variants: {
      variant: {
        default: 'bg-gradient-to-r from-indigo-600 via-blue-600 to-cyan-500 text-white hover:opacity-90 focus-visible:ring-2 focus-visible:ring-blue-500',
        secondary: 'bg-gray-100 text-gray-900 hover:bg-gray-200',
        destructive: 'bg-red-600 text-white hover:bg-red-700 focus-visible:ring-2 focus-visible:ring-red-500',
        outline: 'border border-gray-300 text-gray-900 hover:bg-gray-50 focus-visible:ring-2 focus-visible:ring-gray-400',
        ghost: 'hover:bg-gray-100',
        link: 'text-blue-600 underline-offset-4 hover:underline',
      },
      size: {
        sm: 'h-8 px-3 py-1',
        md: 'h-9 px-4 py-2',
        lg: 'h-10 px-6 py-2',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'md',
    },
  }
)

const Comp = computed(() => props.as ?? 'button')
const klass = computed(() => buttonVariants({ variant: props.variant, size: props.size }))
</script>

<template>
  <component :is="Comp" :class="klass" :disabled="disabled">
    <slot />
  </component>
  
</template>