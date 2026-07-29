import { forwardRef, useId, type ReactNode, type HTMLAttributes } from 'react'
import { useDialogA11y } from '@/hooks/useDialogA11y'
import { cn } from '@/utils/cn'
import { CloseIcon } from './icons'

export interface ModalProps {
  /** Whether the modal is open */
  isOpen: boolean
  /** Callback when modal should close */
  onClose: () => void
  /** Modal title (shown in header) */
  title?: string
  ariaLabel?: string
  /** Modal size */
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full'
  /** Modal content */
  children: ReactNode
  /** Footer content (buttons, etc.) */
  footer?: ReactNode
  /** Close when clicking on overlay */
  closeOnOverlayClick?: boolean
  /** Show close button in header */
  showCloseButton?: boolean
  /** Additional class for modal container */
  className?: string
}

const sizeStyles = {
  sm: 'max-w-sm',      // 384px
  md: 'max-w-lg',      // 512px
  lg: 'max-w-2xl',     // 672px
  xl: 'max-w-4xl',     // 896px
  full: 'max-w-[95vw]',
}

/**
 * Modal dialog component
 *
 * @example
 * <Modal isOpen={isOpen} onClose={() => setIsOpen(false)} title="Settings">
 *   <Modal.Body>
 *     Content here
 *   </Modal.Body>
 *   <Modal.Footer>
 *     <Button variant="secondary" onClick={onClose}>Cancel</Button>
 *     <Button onClick={onSave}>Save</Button>
 *   </Modal.Footer>
 * </Modal>
 */
export const Modal = forwardRef<HTMLDivElement, ModalProps>(
  (
    {
      isOpen,
      onClose,
      title,
      ariaLabel,
      size = 'md',
      children,
      footer,
      closeOnOverlayClick = true,
      showCloseButton = true,
      className,
    },
    ref
  ) => {
    const { dialogRef, onKeyDown } = useDialogA11y<HTMLDivElement>(isOpen, onClose)
    const titleId = useId()

    if (!isOpen) return null

    return (
      <div
        className="fixed inset-0 z-50 flex items-center justify-center"
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? titleId : undefined}
        aria-label={title ? undefined : ariaLabel}
      >
        {/* Overlay */}
        <div
          className="absolute inset-0 bg-black/50 animate-fade-in"
          onClick={closeOnOverlayClick ? onClose : undefined}
        />

        {/* Modal container */}
        <div
          ref={(node) => {
            dialogRef.current = node
            if (typeof ref === 'function') ref(node)
            else if (ref) (ref as React.MutableRefObject<HTMLDivElement | null>).current = node
          }}
          tabIndex={-1}
          onKeyDown={onKeyDown}
          className={cn(
            'relative bg-surface-primary rounded-lg shadow-modal',
            'flex flex-col w-full mx-4',
            'max-h-[90vh] overflow-hidden',
            'animate-slide-up',
            'focus:outline-none',
            sizeStyles[size],
            className
          )}
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header (if title provided) */}
          {(title || showCloseButton) && (
            <div className="px-6 py-4 border-b border-border flex items-center justify-between">
              {title && (
                <h2 id={titleId} className="text-lg font-semibold text-content-primary">
                  {title}
                </h2>
              )}
              {showCloseButton && (
                <button
                  onClick={onClose}
                  className="text-content-muted hover:text-content-secondary transition-colors p-1 -m-1 rounded"
                  aria-label="Close modal"
                >
                  <CloseIcon className="w-5 h-5" />
                </button>
              )}
            </div>
          )}

          {/* Content */}
          {children}

          {/* Footer (if provided) */}
          {footer && (
            <div className="px-6 py-4 border-t border-gray-100 bg-surface-secondary flex items-center justify-end gap-3">
              {footer}
            </div>
          )}
        </div>
      </div>
    )
  }
) as ModalComponent

Modal.displayName = 'Modal'

/* Modal Header (for custom headers) */
interface ModalHeaderProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode
}

const ModalHeader = forwardRef<HTMLDivElement, ModalHeaderProps>(
  ({ className, children, ...props }, ref) => (
    <div
      ref={ref}
      className={cn('px-6 py-4 border-b border-border', className)}
      {...props}
    >
      {children}
    </div>
  )
)
ModalHeader.displayName = 'Modal.Header'

/* Modal Body */
interface ModalBodyProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode
}

const ModalBody = forwardRef<HTMLDivElement, ModalBodyProps>(
  ({ className, children, ...props }, ref) => (
    <div
      ref={ref}
      className={cn('p-6 overflow-y-auto flex-1', className)}
      {...props}
    >
      {children}
    </div>
  )
)
ModalBody.displayName = 'Modal.Body'

/* Modal Footer */
interface ModalFooterProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode
}

const ModalFooter = forwardRef<HTMLDivElement, ModalFooterProps>(
  ({ className, children, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        'px-6 py-4 border-t border-gray-100 bg-surface-secondary flex items-center justify-end gap-3',
        className
      )}
      {...props}
    >
      {children}
    </div>
  )
)
ModalFooter.displayName = 'Modal.Footer'

/* Type for Modal component with subcomponents */
interface ModalComponent
  extends React.ForwardRefExoticComponent<ModalProps & React.RefAttributes<HTMLDivElement>> {
  Header: typeof ModalHeader
  Body: typeof ModalBody
  Footer: typeof ModalFooter
}

// Attach subcomponents
;(Modal as ModalComponent).Header = ModalHeader
;(Modal as ModalComponent).Body = ModalBody
;(Modal as ModalComponent).Footer = ModalFooter
