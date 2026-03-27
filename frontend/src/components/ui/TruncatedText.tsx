import { clsx } from 'clsx'
import { truncateMiddle } from '@/utils/presentation'

interface TruncatedTextProps {
  text: string
  /** Max visible characters before middle ellipsis */
  maxLen?: number
  className?: string
  as?: 'span' | 'div' | 'p'
}

/**
 * Inline truncation with full value in title for hover / SR context (pair with visible label when needed).
 */
export function TruncatedText({ text, maxLen = 48, className, as: Tag = 'span' }: TruncatedTextProps) {
  const display = text.length > maxLen ? truncateMiddle(text, maxLen) : text
  return (
    <Tag className={clsx('min-w-0 font-mono text-xs', className)} title={text}>
      {display}
    </Tag>
  )
}
