import { ReactNode, useState, useRef, useId, isValidElement, cloneElement, type ReactElement } from 'react'
import { clsx } from 'clsx'

interface TooltipProps {
  content: ReactNode
  children: ReactNode
  side?: 'top' | 'bottom' | 'left' | 'right'
  align?: 'start' | 'center' | 'end'
  delay?: number
}

export function Tooltip({
  content,
  children,
  side = 'top',
  align = 'center',
  delay = 240,
}: TooltipProps) {
  const [isVisible, setIsVisible] = useState(false)
  const timeoutRef = useRef<number>()
  const tooltipId = useId()

  const showTooltip = () => {
    timeoutRef.current = window.setTimeout(() => {
      setIsVisible(true)
    }, delay)
  }

  const hideTooltip = () => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
    }
    setIsVisible(false)
  }

  const positions = {
    top: 'bottom-full mb-2.5',
    bottom: 'top-full mt-2.5',
    left: 'right-full mr-2.5 top-1/2 -translate-y-1/2',
    right: 'left-full ml-2.5 top-1/2 -translate-y-1/2',
  }

  const horizontalAlignments = {
    start: 'left-0',
    center: 'left-1/2 -translate-x-1/2',
    end: 'right-0',
  }

  const triggerChild = isValidElement(children)
    ? cloneElement(children as ReactElement<{ 'aria-describedby'?: string }>, {
        'aria-describedby': isVisible ? tooltipId : undefined,
      })
    : (
        <span className="inline-flex" aria-describedby={isVisible ? tooltipId : undefined}>
          {children}
        </span>
      )

  return (
    <div
      className="relative inline-flex"
      onMouseEnter={showTooltip}
      onMouseLeave={hideTooltip}
      onFocus={showTooltip}
      onBlur={hideTooltip}
    >
      {triggerChild}
      {isVisible && (
        <div
          id={tooltipId}
          className={clsx(
            'absolute z-50 max-w-xs rounded-md border border-white/10 bg-slate-950/92 px-3 py-2 text-xs leading-relaxed text-slate-100 shadow-float backdrop-blur-xl animate-fade-in motion-reduce:animate-none',
            positions[side],
            side === 'top' || side === 'bottom' ? horizontalAlignments[align] : undefined
          )}
          role="tooltip"
        >
          {content}
        </div>
      )}
    </div>
  )
}

interface TooltipIconButtonProps {
  icon: ReactNode
  label: string
  onClick?: () => void
  variant?: 'default' | 'ghost' | 'danger'
}

export function TooltipIconButton({
  icon,
  label,
  onClick,
  variant = 'default',
}: TooltipIconButtonProps) {
  const variants = {
    default: 'border-border/70 bg-card/80 text-muted-foreground hover:border-primary/16 hover:text-foreground',
    ghost: 'border-transparent bg-transparent text-muted-foreground hover:bg-accent/65 hover:text-foreground',
    danger: 'border-critical/18 bg-critical/6 text-critical hover:bg-critical/10 hover:text-critical',
  }

  return (
    <Tooltip content={label}>
      <button
        onClick={onClick}
        className={clsx('icon-button', variants[variant])}
        aria-label={label}
      >
        {icon}
      </button>
    </Tooltip>
  )
}
