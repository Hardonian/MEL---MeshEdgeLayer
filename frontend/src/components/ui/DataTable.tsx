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
          <div className="border-b border-border/60 bg-muted/35 px-4 py-3">
            <div className="flex gap-3">
              {columns.map((col) => (
                <div key={col.key} className="skeleton-shimmer h-3 flex-1 rounded-full" />
              ))}
            </div>
          </div>
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="border-b border-border/50 px-4 py-3 last:border-b-0">
              <div className="skeleton-shimmer h-4 rounded-full" />
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (data.length === 0) {
    return (
      <div className="surface-panel px-6 py-8 sm:px-8">
        <div className="text-center">
          <p className="text-sm font-semibold text-foreground">{emptyMessage}</p>
          {emptyDescription && (
            <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">{emptyDescription}</p>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className={clsx('surface-panel overflow-hidden', className)}>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[34rem] border-collapse text-sm">
          <thead className="border-b border-border/70 bg-muted/38">
            <tr>
              {columns.map((column) => (
                <th
                  key={column.key}
                  scope="col"
                  className={clsx(
                    'whitespace-nowrap px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground',
                    column.className
                  )}
                >
                  {column.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-border/55">
            {data.map((item, index) => (
              <tr
                key={String(item[keyField])}
                className={clsx(
                  'transition-colors duration-150 hover:bg-accent/55 focus-within:bg-accent/55',
                  index % 2 === 1 && 'bg-muted/[0.18]'
                )}
              >
                {columns.map((column) => (
                  <td
                    key={column.key}
                    className={clsx(
                      'max-w-[28rem] px-4 py-3.5 align-top text-sm text-foreground',
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
