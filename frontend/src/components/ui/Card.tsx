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
    <div className={clsx('flex flex-col gap-1.5 px-5 py-5 sm:px-6 sm:py-6', className)}>
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
    <h3 className={clsx('font-outfit text-[1.02rem] font-semibold tracking-[-0.02em] text-foreground', className)}>
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
    <p className={clsx('text-sm leading-relaxed text-muted-foreground', className)}>
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
    <div className={clsx('px-5 pb-5 pt-0 sm:px-6 sm:pb-6', className)}>
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
    <div className={clsx('flex items-center px-5 pb-5 pt-0 sm:px-6 sm:pb-6', className)}>
      {children}
    </div>
  )
}
