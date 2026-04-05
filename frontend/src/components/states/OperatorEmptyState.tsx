import { Inbox } from 'lucide-react'

interface OperatorEmptyStateProps {
  title: string
  description: string
  actionNode?: React.ReactNode
}

export function OperatorEmptyState({ title, description, actionNode }: OperatorEmptyStateProps) {
  return (
    <div className="surface-panel surface-panel-muted flex flex-col items-center justify-center border-dashed p-8 text-center">
      <div className="mb-3 text-muted-foreground/30">
        <Inbox className="h-6 w-6" aria-hidden />
      </div>
      <p className="text-mel-xs font-bold text-muted-foreground/50">[EMPTY]</p>
      <h3 className="mt-1 text-mel-sm font-bold uppercase text-foreground">{title}</h3>
      <p className="mt-1 max-w-md text-mel-xs text-muted-foreground">{description}</p>
      {actionNode && <div className="mt-4">{actionNode}</div>}
    </div>
  )
}
