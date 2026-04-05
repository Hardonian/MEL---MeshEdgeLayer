import { ReactNode } from 'react'
import { clsx } from 'clsx'

interface Column<T> {
  key: string
  header: string
  render?: (item: T) => ReactNode
  className?: string
}

interface DataTableProps<T> {
  data: T[]
  columns: Column<T>[]
  keyField: keyof T
  emptyMessage?: string
  emptyDescription?: string
  isLoading?: boolean
  className?: string
}

export function DataTable<T>({
  data,
  columns,
  keyField,
  emptyMessage = 'No data available',
  emptyDescription,
  isLoading,
  className,
}: DataTableProps<T>) {
  if (isLoading) {
    return (
      <div className="surface-panel overflow-hidden">
        <div aria-busy="true" aria-label="Loading table">
          <div className="border-b border-border bg-panel-strong px-3 py-2">
            <div className="flex gap-3">
              {columns.map((col) => (
                <div key={col.key} className="skeleton-shimmer h-2 flex-1" />
              ))}
            </div>
          </div>
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="border-b border-border/40 px-3 py-2 last:border-b-0">
              <div className="skeleton-shimmer h-2.5" />
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (data.length === 0) {
    return (
      <div className="surface-panel px-4 py-5">
        <div className="text-center">
          <p className="text-mel-sm font-bold text-muted-foreground">-- {emptyMessage} --</p>
          {emptyDescription && (
            <p className="mt-1 text-mel-xs text-muted-foreground/60">{emptyDescription}</p>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className={clsx('surface-panel overflow-hidden', className)}>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[34rem] border-collapse text-mel-sm">
          <thead className="border-b border-border bg-panel-strong">
            <tr>
              {columns.map((column) => (
                <th
                  key={column.key}
                  scope="col"
                  className={clsx(
                    'whitespace-nowrap px-3 py-2 text-left font-mono text-mel-xs font-bold uppercase tracking-wide text-muted-foreground',
                    column.className
                  )}
                >
                  {column.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-border/30">
            {data.map((item, index) => (
              <tr
                key={String(item[keyField])}
                className={clsx(
                  'transition-colors duration-75 hover:bg-neon/4 focus-within:bg-neon/4',
                  index % 2 === 1 && 'bg-muted/[0.15]'
                )}
              >
                {columns.map((column) => (
                  <td
                    key={column.key}
                    className={clsx(
                      'max-w-[28rem] px-3 py-2 align-top text-mel-sm text-foreground',
                      column.className
                    )}
                  >
                    {column.render
                      ? column.render(item)
                      : String((item as Record<string, unknown>)[column.key] ?? '')}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
