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
      <div className="rounded-xl border border-border/80 bg-card shadow-sm">
        <div className="animate-pulse" aria-busy="true" aria-label="Loading table">
          <div className="border-b border-border/80 bg-muted/20 p-3">
            <div className="flex gap-3">
              {columns.map((col) => (
                <div key={col.key} className="h-3 flex-1 rounded bg-muted" />
              ))}
            </div>
          </div>
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="border-b border-border/50 p-3 last:border-b-0">
              <div className="h-3 rounded bg-muted/80" />
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (data.length === 0) {
    return (
      <div className="rounded-xl border border-border/80 bg-card p-6 shadow-sm sm:p-8">
        <div className="text-center">
          <p className="text-sm font-medium text-foreground">{emptyMessage}</p>
          {emptyDescription && (
            <p className="mt-1 text-sm text-muted-foreground">{emptyDescription}</p>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className={clsx('rounded-xl border border-border/80 bg-card shadow-sm overflow-hidden', className)}>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[32rem] border-collapse text-sm">
          <thead className="border-b border-border/80 bg-muted/40">
            <tr>
              {columns.map((column) => (
                <th
                  key={column.key}
                  scope="col"
                  className={clsx(
                    'whitespace-nowrap px-3 py-2.5 text-left text-[11px] font-semibold uppercase tracking-wider text-muted-foreground sm:px-4 sm:py-3',
                    column.className
                  )}
                >
                  {column.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-border/60">
            {data.map((item) => (
              <tr
                key={String(item[keyField])}
                className="transition-colors hover:bg-muted/40"
              >
                {columns.map((column) => (
                  <td
                    key={column.key}
                    className={clsx(
                      'max-w-[28rem] px-3 py-2.5 align-top text-sm sm:px-4 sm:py-3',
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
