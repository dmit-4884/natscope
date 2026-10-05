import type { LabelColor } from '@/api/connections'

export const LABEL_COLORS: readonly LabelColor[] = ['gray', 'blue', 'green', 'amber', 'red']

export const LABEL_BADGE_CLASSES: Record<LabelColor, string> = {
  gray: 'bg-gray-600 text-content-inverse',
  blue: 'bg-accent text-content-inverse',
  green: 'bg-green-700 text-content-inverse',
  amber: 'bg-amber-400 text-amber-950',
  red: 'bg-red-600 text-content-inverse',
}

export const LABEL_STRIPE_CLASSES: Record<LabelColor, string> = {
  gray: 'bg-gray-500',
  blue: 'bg-accent',
  green: 'bg-green-600',
  amber: 'bg-amber-400',
  red: 'bg-red-600',
}
