import { type ReactNode, type HTMLAttributes } from 'react'
import { clsx } from 'clsx'
import { Badge, type BadgeVariant } from '../Badge'

/** Panel section header + body — same rhythm as legacy Card inside `MelPanel` (no nested `surface-panel`). */
export function MelPanelSection({
  heading,
  headingClassName,
  description,
  descriptionClassName,
  icon,
  children,
  className,
  headerClassName,
  contentClassName,
  headingLevel = 'h3',
  ...rest
}: Omit<HTMLAttributes<HTMLDivElement>, 'title'> & {
  heading: ReactNode
  headingClassName?: string
  description?: ReactNode
  descriptionClassName?: string
  icon?: ReactNode
  headerClassName?: string
  contentClassName?: string
  headingLevel?: 'h2' | 'h3' | 'h4'
}) {
  const HeadingTag = headingLevel
  return (
    <div className={clsx(className)} {...rest}>
      <div className={clsx('mel-chrome-title', headerClassName)}>
        <div className="flex flex-col gap-0.5 py-0.5">
          <HeadingTag
            className={clsx(
              'text-mel-sm font-bold uppercase tracking-[0.08em] text-foreground',
              headingClassName,
            )}
          >
            {icon ? (
              <span className="inline-flex items-center gap-2">
                {icon}
                {heading}
              </span>
            ) : (
              heading
            )}
          </HeadingTag>
          {description != null && description !== '' && (
            <p className={clsx('text-mel-xs text-muted-foreground mt-1', descriptionClassName)}>{description}</p>
          )}
        </div>
      </div>
      <div className={clsx('px-3 pb-3 pt-2', contentClassName)}>{children}</div>
    </div>
  )
}

/** Repeated dense list / evidence row — shared rhythm (not a full callout). */
export function MelDenseRow({
  className,
  tone = 'default',
  ...rest
}: HTMLAttributes<HTMLDivElement> & { tone?: 'default' | 'warning' | 'muted' }) {
  const toneCls =
    tone === 'warning'
      ? 'border-signal-partial/28 bg-signal-partial/[0.06]'
      : tone === 'muted'
        ? 'border-border/50 bg-muted/15'
        : 'border-border/50 bg-card/40'
  return (
    <div
      className={clsx('rounded-[var(--radius)] border px-3 py-2 text-xs', toneCls, className)}
      {...rest}
    />
  )
}

/** Framed panel — `surface-panel` + optional chrome modifiers */
export function MelPanel({
  children,
  className,
  muted,
  strong,
  ...rest
}: HTMLAttributes<HTMLDivElement> & { muted?: boolean; strong?: boolean }) {
  return (
    <div
      className={clsx(
        'surface-panel text-card-foreground',
        muted && 'surface-panel-muted',
        strong && 'surface-panel-strong',
        className,
      )}
      {...rest}
    >
      {children}
    </div>
  )
}

const insetToneClasses: Record<
  'default' | 'observed' | 'degraded' | 'critical' | 'info' | 'warning' | 'stale',
  string
> = {
  default: 'border-border/80 bg-muted/25',
  observed: 'border-signal-observed/25 bg-signal-observed/5',
  degraded: 'border-signal-degraded/30 bg-signal-degraded/[0.06]',
  critical: 'border-signal-critical/30 bg-signal-critical/[0.06]',
  info: 'border-signal-observed/20 bg-signal-observed/5',
  warning: 'border-signal-partial/28 bg-signal-partial/[0.06]',
  stale: 'border-signal-stale/30 bg-signal-stale/[0.06]',
}

/** Inset / remediation / callout — canonical `mel-panel-inset` + optional truth tone */
export function MelPanelInset({
  children,
  className,
  tone = 'default',
  ...rest
}: HTMLAttributes<HTMLDivElement> & { tone?: keyof typeof insetToneClasses }) {
  return (
    <div className={clsx('mel-panel-inset', insetToneClasses[tone], className)} {...rest}>
      {children}
    </div>
  )
}

/** Page section with optional eyebrow (mono label) and heading */
export function MelPageSection({
  id,
  eyebrow,
  title,
  titleClassName,
  description,
  children,
  className,
  headingLevel: H = 'h2',
}: {
  id?: string
  eyebrow?: string
  title?: ReactNode
  titleClassName?: string
  description?: ReactNode
  children?: ReactNode
  className?: string
  headingLevel?: 'h2' | 'h3'
}) {
  const headingId = id && title != null && title !== '' ? `${id}-heading` : undefined
  return (
    <section id={id} className={clsx('space-y-3', className)} aria-labelledby={headingId}>
      {(eyebrow || title || description) && (
        <header className="space-y-1">
          {eyebrow && <p className="mel-label">{eyebrow}</p>}
          {title != null && title !== '' && (
            <H
              id={headingId}
              className={clsx(
                'text-mel-sm font-semibold uppercase tracking-[0.18em] text-foreground',
                titleClassName,
              )}
            >
              {title}
            </H>
          )}
          {description && <p className="text-mel-sm text-muted-foreground">{description}</p>}
        </header>
      )}
      {children}
    </section>
  )
}

/** Segmented control shell — pair with `MelSegmentItem`. Use `radiogroupLabel` when items are `role="radio"`. */
export function MelSegment({
  children,
  className,
  label,
  radiogroupLabel,
  ...rest
}: HTMLAttributes<HTMLDivElement> & { label: string; radiogroupLabel?: string }) {
  const shell = (
    <>
      <span className="mel-label shrink-0 text-muted-foreground">{label}</span>
      <div
        className="mel-segment flex flex-wrap"
        role={radiogroupLabel ? undefined : 'group'}
        aria-label={radiogroupLabel ? undefined : label}
      >
        {children}
      </div>
    </>
  )
  if (radiogroupLabel) {
    return (
      <div
        className={clsx('flex flex-wrap items-center gap-2', className)}
        role="radiogroup"
        aria-label={radiogroupLabel}
        {...rest}
      >
        {shell}
      </div>
    )
  }
  return (
    <div className={clsx('flex flex-wrap items-center gap-2', className)} {...rest}>
      {shell}
    </div>
  )
}

export function MelSegmentItem({
  active,
  children,
  className,
  ...rest
}: HTMLAttributes<HTMLButtonElement> & { active?: boolean }) {
  return (
    <button
      type="button"
      className={clsx(
        'mel-segment-item',
        active ? 'mel-segment-item-active' : 'mel-segment-item-inactive',
        className,
      )}
      {...rest}
    >
      {children}
    </button>
  )
}

/** Compact metric block — sans label, mono value */
export function MelStat({
  label,
  value,
  description,
  className,
}: {
  label: string
  value: ReactNode
  description?: ReactNode
  className?: string
}) {
  return (
    <div className={clsx('min-w-0', className)}>
      <p className="mel-label">{label}</p>
      <p className="mt-1 font-data text-mel-metric font-bold tabular-nums text-neon">{value}</p>
      {description && <p className="mt-1 text-mel-xs text-muted-foreground">{description}</p>}
    </div>
  )
}

const truthToBadge: Record<string, BadgeVariant> = {
  observed: 'observed',
  inferred: 'inferred',
  stale: 'stale',
  frozen: 'frozen',
  unsupported: 'unsupported',
  degraded: 'degraded',
  partial: 'partial',
  complete: 'complete',
  critical: 'critical',
  live: 'success',
  unknown: 'secondary',
}

export type MelTruthSemantic = keyof typeof truthToBadge

/** Map canonical truth vocabulary to bracket badges (not a substitute for explanatory copy). */
export function MelTruthBadge({
  semantic,
  children,
  className,
}: {
  semantic: string
  children: ReactNode
  className?: string
}) {
  const variant = truthToBadge[semantic] ?? 'outline'
  return (
    <Badge variant={variant} className={className}>
      {children}
    </Badge>
  )
}
