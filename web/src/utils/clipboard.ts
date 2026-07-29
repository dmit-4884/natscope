import { toast } from '@/utils/toast'

export async function copyText(text: string, successMessage = 'Copied to clipboard'): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    toast.success(successMessage, { duration: 2000 })
    return true
  } catch {
    toast.error('Failed to copy')
    return false
  }
}
