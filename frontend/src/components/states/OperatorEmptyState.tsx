import { Inbox } from 'lucide-react'

interface OperatorEmptyStateProps {
  title: string
  description: string
  actionNode?: React.ReactNode
}

export function OperatorEmptyState({ title, description, actionNode }: OperatorEmptyStateProps) {
  return (
    <div className="surface-panel surface-panel-muted flex flex-col items-center justify-center rounded-[1.1rem] border-dashed p-10 text-center">
      <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-[1.25rem] border border-border/70 bg-card/70 text-primary shadow-inset">
        <Inbox className="h-7 w-7 opacity-80" aria-hidden />
      </div>
      <h3 className="font-outfit text-lg font-semibold tracking-[-0.02em] text-foreground">{title}</h3>
      <p className="mt-1 max-w-md text-sm leading-relaxed text-muted-foreground">{description}</p>
      {actionNode && <div className="mt-6">{actionNode}</div>}
    </div>
  )
}
