<script setup lang="ts">
import { cva } from 'class-variance-authority'
import { computed } from 'vue'

type Variant = 'default' | 'secondary' | 'destructive' | 'outline' | 'ghost' | 'link'
type Size = 'sm' | 'md' | 'lg'

const props = defineProps<{
  variant?: Variant
  size?: Size
  as?: string
  disabled?: boolean
}>()

const buttonVariants = cva(
  'inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default: 'bg-black text-white hover:bg-gray-800',
        secondary: 'bg-gray-100 text-gray-900 hover:bg-gray-200',
        destructive: 'bg-red-600 text-white hover:bg-red-700',
        outline: 'border border-gray-300 hover:bg-gray-50',
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