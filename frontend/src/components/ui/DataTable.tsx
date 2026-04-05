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
          <div className="border-b border-border/50 bg-muted/25 px-3 py-2.5">
            <div className="flex gap-3">
              {columns.map((col) => (
                <div key={col.key} className="skeleton-shimmer h-2.5 flex-1 rounded-sm" />
              ))}
            </div>
          </div>
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="border-b border-border/40 px-3 py-2.5 last:border-b-0">
              <div className="skeleton-shimmer h-3 rounded-sm" />
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (data.length === 0) {
    return (
      <div className="surface-panel px-5 py-6">
        <div className="text-center">
          <p className="font-mono text-mel-sm font-semibold text-foreground">{emptyMessage}</p>
          {emptyDescription && (
            <p className="mt-1 prose-body text-mel-xs text-muted-foreground">{emptyDescription}</p>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className={clsx('surface-panel overflow-hidden', className)}>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[34rem] border-collapse font-mono text-mel-sm">
          <thead className="border-b border-border/60 bg-muted/20">
            <tr>
              {columns.map((column) => (
                <th
                  key={column.key}
                  scope="col"
                  className={clsx(
                    'whitespace-nowrap px-3 py-2.5 text-left mel-label',
                    column.className
                  )}
                >
                  {column.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-border/40">
            {data.map((item, index) => (
              <tr
                key={String(item[keyField])}
                className={clsx(
                  'transition-colors duration-100 hover:bg-accent/40 focus-within:bg-accent/40',
                  index % 2 === 1 && 'bg-muted/[0.12]'
                )}
              >
                {columns.map((column) => (
                  <td
                    key={column.key}
                    className={clsx(
                      'max-w-[28rem] px-3 py-2.5 align-top text-mel-sm text-foreground',
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
