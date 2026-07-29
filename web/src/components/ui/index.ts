/**
 * UI Component Library
 *
 * This module exports all reusable UI components for the Natscope application.
 * Components are designed to be consistent with the existing visual style while providing
 * a unified API for common UI patterns.
 *
 * @example
 * import { Button, Input, Card, Modal, Badge } from '@/components/ui'
 */

// Core components
export { Button, type ButtonProps } from './Button'
export { Input, type InputProps } from './Input'
export { Modal, type ModalProps } from './Modal'
export { DestructiveConfirm, type DestructiveConfirmProps } from './DestructiveConfirm'

// Feedback components
export { Alert, type AlertProps } from './Alert'
export { Badge, type BadgeProps } from './Badge'
export { Spinner, type SpinnerProps } from './Spinner'
export { SkeletonRows, type SkeletonRowsProps } from './Skeleton'
export { EmptyState, type EmptyStateProps } from './EmptyState'
export { QueryErrorState, type QueryErrorStateProps } from './QueryErrorState'
// Form components
export { ImmutableField } from './ImmutableField'
export { Select, type SelectProps } from './Select'
export { Dropdown, type DropdownProps } from './Dropdown'
export { OverflowMenu, type OverflowMenuItem } from './OverflowMenu'
export { SearchableSelect, type SearchableSelectProps, type SearchableSelectOption } from './SearchableSelect'
export { SearchInput, type SearchInputProps } from './SearchInput'
export { Toggle, type ToggleProps } from './Toggle'
export { JsonEditor } from './JsonEditor'

// Navigation components
export { Tabs, type TabsProps } from './Tabs'
export { tabPanelProps } from './tabPanel'

// Action components
export { CopyButton, type CopyButtonProps } from './CopyButton'
export { RowActionButton, type RowActionButtonProps } from './RowActionButton'

// Data display components
export { DataTable, type DataTableProps, type DataTableColumn } from './DataTable'

// Status components
export { FilterChip, FilterChipsGroup, type FilterChipProps, type FilterChipsGroupProps } from './FilterChip'

export {
  ChevronDownIcon,
  ChevronUpIcon,
  CloseIcon,
  PlusIcon,
  TrashIcon,
  PencilIcon,
  RefreshIcon,
  CheckIcon,
  WarningIcon,
  PauseIcon,
  CopyIcon,
  ClipboardIcon,
  DocumentIcon,
  SearchIcon,
  DownloadIcon,
  UploadIcon,
  InfoIcon,
  BoltIcon,
  LockClosedIcon,
  UsersIcon,
  EyeIcon,
  EyeOffIcon,
  DotsHorizontalIcon,
} from './icons'
