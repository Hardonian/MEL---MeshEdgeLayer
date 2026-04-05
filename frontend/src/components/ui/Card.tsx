import { ReactNode } from 'react'
import { clsx } from 'clsx'

interface CardProps {
  children: ReactNode
  className?: string
  id?: string
}

export function Card({ children, className, id }: CardProps) {
  return (
    <div
      id={id}
      className={clsx('surface-panel text-card-foreground', className)}
    >
      {children}
    </div>
  )
}

interface CardHeaderProps {
  children: ReactNode
  className?: string
}

export function CardHeader({ children, className }: CardHeaderProps) {
  return (
    <div className={clsx('flex flex-col gap-1 px-4 py-4 sm:px-5', className)}>
      {children}
    </div>
  )
}

interface CardTitleProps {
  children: ReactNode
  className?: string
}

export function CardTitle({ children, className }: CardTitleProps) {
  return (
    <h3 className={clsx('font-display text-sm font-bold tracking-tight text-foreground', className)}>
      {children}
    </h3>
  )
}

interface CardDescriptionProps {
  children: ReactNode
  className?: string
}

export function CardDescription({ children, className }: CardDescriptionProps) {
  return (
    <p className={clsx('prose-body text-mel-sm text-muted-foreground', className)}>
      {children}
    </p>
  )
}

interface CardContentProps {
  children: ReactNode
  className?: string
}

export function CardContent({ children, className }: CardContentProps) {
  return (
    <div className={clsx('px-4 pb-4 pt-0 sm:px-5 sm:pb-5', className)}>
      {children}
    </div>
  )
}

interface CardFooterProps {
  children: ReactNode
  className?: string
}

export function CardFooter({ children, className }: CardFooterProps) {
  return (
    <div className={clsx('flex items-center px-4 pb-4 pt-0 sm:px-5 sm:pb-5', className)}>
      {children}
    </div>
  )
}
