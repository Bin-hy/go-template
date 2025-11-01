import { ref } from 'vue'

export function useAsync<T extends (...args: any[]) => Promise<any>>(fn: T) {
  const loading = ref(false)
  const error = ref<unknown>(null)

  const run = async (...args: Parameters<T>) => {
    loading.value = true
    error.value = null
    try {
      const res = await fn(...args)
      return res
    } catch (e) {
      error.value = e
      throw e
    } finally {
      loading.value = false
    }
  }

  return { loading, error, run }
}