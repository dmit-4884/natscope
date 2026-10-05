import { cn } from '@/utils/cn'

export type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'ghost' | 'link'
export type ButtonSize = 'sm' | 'md' | 'lg'

const variantStyles = {
  primary: [
    'bg-accent text-content-inverse',
    'hover:bg-accent-hover',
    'active:bg-blue-800',
    'disabled:bg-gray-300 disabled:text-content-tertiary',
  ].join(' '),
  secondary: [
    'bg-surface-primary text-gray-700 border border-border-strong',
    'hover:bg-surface-secondary',
    'active:bg-surface-tertiary',
    'disabled:bg-surface-tertiary disabled:text-content-muted',
  ].join(' '),
  danger: [
    'bg-red-600 text-content-inverse',
    'hover:bg-red-700',
    'active:bg-red-800',
    'disabled:bg-gray-300 disabled:text-content-tertiary',
  ].join(' '),
  ghost: [
    'text-content-secondary bg-transparent',
    'hover:bg-surface-tertiary hover:text-content-primary',
    'active:bg-surface-hover',
    'disabled:text-content-muted disabled:bg-transparent',
  ].join(' '),
  link: [
    'text-accent bg-transparent underline-offset-4',
    'hover:text-accent-text hover:underline',
    'active:text-blue-800',
    'disabled:text-content-muted disabled:no-underline',
  ].join(' '),
}

const sizeStyles = {
  sm: 'px-2.5 py-1.5 text-xs gap-1.5',
  md: 'px-4 py-2 text-sm gap-2',
  lg: 'px-6 py-3 text-base gap-2.5',
}

export function buttonClassName(variant: ButtonVariant = 'primary', size: ButtonSize = 'md') {
  return cn(
    'inline-flex items-center justify-center font-medium rounded-md',
    'transition-colors duration-150',
    'focus:outline-none focus:ring-2 focus:ring-border-focus focus:ring-offset-2',
    'disabled:cursor-not-allowed',
    'active:scale-[0.98]',
    variantStyles[variant],
    sizeStyles[size],
  )
}
