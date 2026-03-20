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
      <div className="rounded-xl border bg-card">
        <div className="animate-pulse">
          <div className="border-b p-4">
            <div className="flex gap-4">
              {columns.map((col) => (
                <div key={col.key} className="h-4 flex-1 bg-muted rounded" />
              ))}
            </div>
          </div>
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="border-b p-4">
              <div className="h-4 bg-muted rounded" />
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (data.length === 0) {
    return (
      <div className="rounded-xl border bg-card p-8">
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
    <div className={clsx('rounded-xl border bg-card overflow-hidden', className)}>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="border-b bg-muted/30">
              {columns.map((column) => (
                <th
                  key={column.key}
                  className={clsx(
                    'px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-muted-foreground',
                    column.className
                  )}
                >
                  {column.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.map((item, index) => (
              <tr
                key={String(item[keyField])}
                className={clsx(
                  'border-b transition-colors hover:bg-muted/50',
                  index === data.length - 1 && 'border-b-0'
                )}
              >
                {columns.map((column) => (
                  <td
                    key={column.key}
                    className={clsx('px-4 py-3 text-sm', column.className)}
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
