import { Inbox } from 'lucide-react'

interface OperatorEmptyStateProps {
  title: string
  description: string
  actionNode?: React.ReactNode
}

export function OperatorEmptyState({ title, description, actionNode }: OperatorEmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border/80 bg-muted/10 p-10 text-center">
      <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-muted/50 text-muted-foreground">
        <Inbox className="h-7 w-7 opacity-70" aria-hidden />
      </div>
      <h3 className="text-sm font-semibold text-foreground">{title}</h3>
      <p className="mt-1 max-w-md text-sm text-muted-foreground">{description}</p>
      {actionNode && <div className="mt-6">{actionNode}</div>}
    </div>
  )
}