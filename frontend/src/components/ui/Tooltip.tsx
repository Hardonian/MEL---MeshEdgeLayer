import { ReactNode, useState, useRef } from 'react'
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
  delay = 300 
}: TooltipProps) {
  const [isVisible, setIsVisible] = useState(false)
  const timeoutRef = useRef<number>()

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
    top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
    bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
    left: 'right-full top-1/2 -translate-y-1/2 mr-2',
    right: 'left-full top-1/2 -translate-y-1/2 ml-2',
  }

  const alignments = {
    start: 'left-0',
    center: 'left-1/2 -translate-x-1/2',
    end: 'right-0',
  }

  return (
    <div 
      className="relative inline-flex"
      onMouseEnter={showTooltip}
      onMouseLeave={hideTooltip}
      onFocus={showTooltip}
      onBlur={hideTooltip}
    >
      {children}
      {isVisible && (
        <div
          className={clsx(
            'absolute z-50 px-3 py-1.5 text-sm rounded-md bg-foreground text-background shadow-md whitespace-nowrap animate-fade-in',
            positions[side],
            align === 'center' ? alignments[align] : (side === 'top' || side === 'bottom' ? alignments[align] : 'translate-y-0')
          )}
          role="tooltip"
        >
          {content}
        </div>
      )}
    </div>
  )
}

// Icon button with tooltip
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
  variant = 'default' 
}: TooltipIconButtonProps) {
  const variants = {
    default: 'hover:bg-accent',
    ghost: 'hover:bg-muted',
    danger: 'hover:bg-critical/10 text-critical hover:text-critical',
  }

  return (
    <Tooltip content={label}>
      <button
        onClick={onClick}
        className={clsx(
          'p-2 rounded-md transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
          variants[variant]
        )}
        aria-label={label}
      >
        {icon}
      </button>
    </Tooltip>
  )
}
