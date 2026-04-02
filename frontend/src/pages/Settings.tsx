import type { ReactNode } from 'react'
import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { KeyboardShortcuts } from '@/components/ui/HelpMenu'
import { useConsoleThemePreference } from '@/hooks/useConsoleThemePreference'
import { useVersionInfo } from '@/hooks/useVersionInfo'
import { useStatus } from '@/hooks/useApi'
import type { PlatformPosture } from '@/types/api'
import {
  Server,
  Info,
  ExternalLink,
  Lock,
  Database,
  Wifi,
  Shield,
  Terminal,
  BookOpen,
  Wrench,
  Monitor,
  Cpu,
  Radio,
} from 'lucide-react'

function nested(root: Record<string, unknown> | undefined, path: string): unknown {
  if (!root) return undefined
  return path.split('.').reduce<unknown>((acc, part) => {
    if (!acc || typeof acc !== 'object' || Array.isArray(acc)) return undefined
    return (acc as Record<string, unknown>)[part]
  }, root)
}

function stringifyValue(value: unknown): string | null {
  if (value == null) return null
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  try {
    return JSON.stringify(value)
  } catch {
    return '[unserializable]'
  }
}

export function SettingsPage() {
  const version = useVersionInfo()
  const status = useStatus()
  const { preference, setPreference } = useConsoleThemePreference()
  const [configInspect, setConfigInspect] = useState<{ values?: Record<string, unknown>; fingerprint?: string } | null>(null)
  const [configError, setConfigError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const res = await fetch('/api/v1/config/inspect')
        if (!res.ok) {
          throw new Error(`HTTP ${res.status}`)
        }
        const data = (await res.json()) as { values?: Record<string, unknown>; fingerprint?: string }
        if (!cancelled) {
          setConfigInspect(data)
          setConfigError(null)
        }
      } catch (e) {
        if (!cancelled) {
          setConfigError(e instanceof Error ? e.message : 'config inspect unavailable')
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  const effectiveRows = useMemo<Array<{ key: string; defaultValue: string; effective: string | null }>>(
    () => [
      { key: 'bind.api', defaultValue: '"127.0.0.1:8080"', effective: stringifyValue(nested(configInspect?.values, 'bind.api')) },
      { key: 'bind.metrics', defaultValue: '""', effective: stringifyValue(nested(configInspect?.values, 'bind.metrics')) },
      { key: 'auth.enabled', defaultValue: 'false', effective: stringifyValue(nested(configInspect?.values, 'auth.enabled')) },
      { key: 'auth.ui_user', defaultValue: '"admin"', effective: stringifyValue(nested(configInspect?.values, 'auth.ui_user')) },
      { key: 'storage.database_path', defaultValue: '"./data/mel.db"', effective: stringifyValue(nested(configInspect?.values, 'storage.database_path')) },
      { key: 'privacy.redact_exports', defaultValue: 'true', effective: stringifyValue(nested(configInspect?.values, 'privacy.redact_exports')) },
      { key: 'privacy.map_reporting_allowed', defaultValue: 'false', effective: stringifyValue(nested(configInspect?.values, 'privacy.map_reporting_allowed')) },
      { key: 'features.google_maps_in_topology_ui', defaultValue: 'false', effective: stringifyValue(nested(configInspect?.values, 'features.google_maps_in_topology_ui')) },
      { key: 'features.google_maps_api_key_env', defaultValue: '""', effective: stringifyValue(nested(configInspect?.values, 'features.google_maps_api_key_env')) },
      { key: 'features.metrics', defaultValue: 'false', effective: stringifyValue(nested(configInspect?.values, 'features.metrics')) },
    ],
    [configInspect?.values]
  )

  const v = version.data
  const versionLine = v?.version ?? null
  const goLine = v?.go_version ?? '—'
  const schemaOk =
    typeof v?.schema_matches_binary === 'boolean' ? (v.schema_matches_binary ? 'Yes' : 'No') : '—'
  const dbActual = v?.db_actual_version ?? '—'

  return (
    <div className="space-y-6">
      <PageHeader
        title="Settings"
        description="Console preferences (this browser), read-only configuration reference, and links to operator documentation."
      />

      <RuntimeTruthStrip version={v} status={status.data} versionLoading={version.loading} statusLoading={status.loading} />

      {/* Quick Access */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <QuickAccessCard
          icon={<Wrench className="h-5 w-5" />}
          title="Configuration"
          description="Jump to config keys in this page"
          href="#config"
        />
        <QuickAccessCard
          icon={<Database className="h-5 w-5" />}
          title="Storage"
          description="Data directory and retention keys"
          href="#storage"
        />
        <QuickAccessCard
          icon={<Shield className="h-5 w-5" />}
          title="Privacy"
          description="Privacy audit and findings"
          to="/privacy"
        />
        <QuickAccessCard
          icon={<Terminal className="h-5 w-5" />}
          title="CLI"
          description="Command-line reference (repo)"
          href="#cli"
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Monitor className="h-5 w-5" />
            Console preferences
          </CardTitle>
          <CardDescription>
            Stored only in this browser. Does not change MEL configuration on disk or server behavior.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <fieldset className="space-y-2">
            <legend className="text-sm font-medium text-foreground">Appearance</legend>
            <p className="text-sm text-muted-foreground">
              Chooses light or dark styling for the web UI. Use &quot;Match system&quot; to follow OS theme.
            </p>
            <div className="flex flex-wrap gap-2">
              {(
                [
                  { value: 'system' as const, label: 'Match system' },
                  { value: 'light' as const, label: 'Light' },
                  { value: 'dark' as const, label: 'Dark' },
                ] as const
              ).map((opt) => (
                <button
                  key={opt.value}
                  type="button"
                  onClick={() => setPreference(opt.value)}
                  className={`rounded-md border px-3 py-1.5 text-sm font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ${
                    preference === opt.value
                      ? 'border-primary bg-primary/10 text-foreground'
                      : 'border-border bg-background text-muted-foreground hover:bg-muted hover:text-foreground'
                  }`}
                  aria-pressed={preference === opt.value}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </fieldset>

          <div className="rounded-lg border border-border bg-muted/30 p-4">
            <p className="text-sm font-medium text-foreground">Keyboard shortcuts</p>
            <p className="mt-1 text-xs text-muted-foreground">
              Use the Help menu in the header for documentation links and the full list.
            </p>
            <div className="mt-3">
              <KeyboardShortcuts />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Configuration Reference */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Server className="h-5 w-5" />
            Effective running configuration
          </CardTitle>
          <CardDescription>
            Values from <code className="rounded bg-muted px-1 font-mono text-xs">GET /api/v1/config/inspect</code>. This reflects runtime-loaded config with redaction.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {configError ? (
            <AlertCard
              variant="warning"
              title="Effective config unavailable"
              description={`Could not read /api/v1/config/inspect (${configError}). Showing documented defaults only.`}
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="text-left text-muted-foreground">
                    <th className="py-2 pr-3">Key</th>
                    <th className="py-2 pr-3">Documented default</th>
                    <th className="py-2 pr-3">Effective value</th>
                    <th className="py-2">Truth source</th>
                  </tr>
                </thead>
                <tbody>
                  {effectiveRows.map(({ key, defaultValue, effective }) => (
                    <tr key={key} className="border-t border-border/60">
                      <td className="py-2 pr-3 font-mono">{key}</td>
                      <td className="py-2 pr-3 text-muted-foreground">{defaultValue}</td>
                      <td className="py-2 pr-3 font-mono">{effective ?? 'unavailable'}</td>
                      <td className="py-2">
                        {effective == null ? <Badge variant="warning">unreadable</Badge> : <Badge variant="success">runtime loaded</Badge>}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {configInspect?.fingerprint && (
            <p className="text-xs text-muted-foreground">Config fingerprint: <code>{configInspect.fingerprint}</code></p>
          )}
        </CardContent>
      </Card>

      <Card id="config">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Server className="h-5 w-5" />
            Configuration reference
          </CardTitle>
          <CardDescription>
            Keys and defaults as documented for MEL; values on the running instance are read from the config file on disk.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-8">
            {/* Network */}
            <section id="network" aria-labelledby="settings-network-heading">
              <h3 id="settings-network-heading" className="mb-3 flex items-center gap-2 text-sm font-semibold">
                <Wifi className="h-4 w-4 text-muted-foreground" aria-hidden />
                Network binding
              </h3>
              <div className="grid gap-3 sm:grid-cols-2">
                <ConfigItem
                  name="bind.api"
                  type="string"
                  default='"127.0.0.1:8080"'
                  description="HTTP API listen address"
                />
                <ConfigItem
                  name="bind.metrics"
                  type="string"
                  default='""'
                  description="Prometheus metrics endpoint"
                />
              </div>
            </section>

            {/* Auth */}
            <section id="auth" aria-labelledby="settings-auth-heading">
              <h3 id="settings-auth-heading" className="mb-3 flex items-center gap-2 text-sm font-semibold">
                <Lock className="h-4 w-4 text-muted-foreground" aria-hidden />
                Authentication
              </h3>
              <div className="grid gap-3 sm:grid-cols-2">
                <ConfigItem name="auth.enabled" type="bool" default="false" description="Enable basic auth for UI" />
                <ConfigItem name="auth.ui_user" type="string" default='"admin"' description="Username for UI access" />
              </div>
            </section>

            {/* Storage */}
            <section id="storage" aria-labelledby="settings-storage-heading">
              <h3 id="settings-storage-heading" className="mb-3 flex items-center gap-2 text-sm font-semibold">
                <Database className="h-4 w-4 text-muted-foreground" aria-hidden />
                Storage
              </h3>
              <div className="grid gap-3 sm:grid-cols-2">
                <ConfigItem
                  name="storage.data_dir"
                  type="string"
                  default='"./data"'
                  description="Directory for MEL data"
                />
                <ConfigItem
                  name="storage.database_path"
                  type="string"
                  default='"./data/mel.db"'
                  description="SQLite database path"
                />
                <ConfigItem name="retention.messages_days" type="int" default="30" description="Message retention period" />
                <ConfigItem name="retention.audit_days" type="int" default="90" description="Audit log retention" />
              </div>
            </section>

            {/* Privacy */}
            <section id="privacy-keys" aria-labelledby="settings-privacy-keys-heading">
              <h3 id="settings-privacy-keys-heading" className="mb-3 flex items-center gap-2 text-sm font-semibold">
                <Shield className="h-4 w-4 text-muted-foreground" aria-hidden />
                Privacy (config keys)
              </h3>
              <div className="grid gap-3 sm:grid-cols-2">
                <ConfigItem
                  name="privacy.store_precise_positions"
                  type="bool"
                  default="false"
                  description="Store exact GPS coordinates"
                />
                <ConfigItem
                  name="privacy.mqtt_encryption_required"
                  type="bool"
                  default="true"
                  description="Require TLS for MQTT"
                />
                <ConfigItem name="privacy.map_reporting_allowed" type="bool" default="false" description="Allow map reporting" />
                <ConfigItem name="privacy.redact_exports" type="bool" default="true" description="Redact sensitive data in exports" />
              </div>
            </section>

            {/* Features */}
            <section id="features" aria-labelledby="settings-features-heading">
              <h3 id="settings-features-heading" className="mb-3 text-sm font-semibold">
                Features
              </h3>
              <div className="grid gap-3 sm:grid-cols-2">
                <ConfigItem name="features.web_ui" type="bool" default="true" description="Enable built-in web UI" />
                <ConfigItem name="features.metrics" type="bool" default="false" description="Enable Prometheus metrics" />
              </div>
            </section>
          </div>
        </CardContent>
      </Card>

      {/* System Info */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Info className="h-5 w-5" />
            Running instance
          </CardTitle>
          <CardDescription>
            From <code className="rounded bg-muted px-1 font-mono text-xs">GET /api/v1/version</code> on the connected backend.
            {version.loading && ' Loading…'}
            {version.error && (
              <span className="mt-1 block text-critical">
                Could not load version: {version.error}. Is the API reachable?
              </span>
            )}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <InfoCard
              title="MEL version"
              value={versionLine ?? (version.loading ? '…' : '—')}
              description="Binary semantic version"
            />
            <InfoCard title="Go runtime" value={goLine} description="Toolchain reported by the server" />
            <InfoCard title="API surface" value="v1" description="REST paths under /api/v1" />
            <InfoCard title="UI stack" value="React" description="This operator console" />
            <InfoCard title="DB schema (actual)" value={dbActual} description="Schema version in the open database" />
            <InfoCard title="Schema matches binary" value={schemaOk} description="Migration level vs embedded expectation" />
          </div>
        </CardContent>
      </Card>

      {/* Documentation Links */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BookOpen className="h-5 w-5" />
            Documentation
          </CardTitle>
          <CardDescription>
            Paths under <code className="rounded bg-muted px-1 font-mono text-xs">/docs/...</code> are served when the UI is built with
            static documentation assets. In local Vite dev, those URLs may 404 unless you configure static hosting separately.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2">
            <DocLink
              title="Configuration guide"
              description="Full configuration reference"
              href="/docs/ops/configuration.md"
            />
            <DocLink
              title="CLI reference"
              description="Command-line interface"
              href="/docs/ops/cli-reference.md"
              id="cli"
            />
            <DocLink title="First 10 minutes" description="Operator quick start" href="/docs/ops/first-10-minutes.md" />
            <DocLink title="API reference" description="Endpoints and schemas" href="/docs/ops/api-reference.md" />
            <DocLink title="Transport matrix" description="Protocols and capabilities" href="/docs/ops/transport-matrix.md" />
            <DocLink title="Troubleshooting" description="Common issues" href="/docs/ops/troubleshooting.md" />
          </div>
        </CardContent>
      </Card>

      {/* Platform posture — effective running truth */}
      {v?.platform_posture && <PlatformPostureCard posture={v.platform_posture} />}

      {/* Note about config editing */}
      <AlertCard
        variant="info"
        title="Configuration is file-based"
        description="MEL reads settings from a JSON config file on the host. Edit that file and restart the process for changes to apply. This UI does not persist server configuration."
      />
    </div>
  )
}

function RuntimeTruthStrip({
  version,
  status,
  versionLoading,
  statusLoading,
}: {
  version: import('@/types/api').VersionResponse | null | undefined
  status: import('@/types/api').StatusResponse | null | undefined
  versionLoading: boolean
  statusLoading: boolean
}) {
  const topo = version?.topology_model_enabled === true
  const transports = status?.transports ?? []
  const anyLive = transports.some((t) => t.effective_state === 'connected' || t.runtime_state === 'live')
  const anyConfigured = transports.length > 0
  const schemaOk = version?.schema_matches_binary === true
  const fp = version?.config_canonical_fingerprint

  return (
    <Card className="border-primary/15 bg-muted/20" data-testid="settings-runtime-truth-strip">
      <CardHeader className="pb-2">
        <CardTitle className="text-base flex items-center gap-2">
          <Monitor className="h-4 w-4" aria-hidden />
          Runtime truth at a glance
        </CardTitle>
        <CardDescription>
          What the connected API reports now — distinct from documented defaults. “Configured” is not the same as “yielding fresh evidence”.
        </CardDescription>
      </CardHeader>
      <CardContent className="pt-0">
        {(versionLoading || statusLoading) && <p className="text-xs text-muted-foreground">Loading runtime endpoints…</p>}
        {!versionLoading && !statusLoading && (
          <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4 text-xs">
            <div className="rounded-lg border border-border/70 bg-background/80 p-2.5">
              <p className="font-medium text-foreground">Topology model</p>
              <p className="mt-1 text-muted-foreground">
                {version?.topology_model_enabled === undefined
                  ? 'Unknown (older server — upgrade to expose topology_model_enabled)'
                  : topo
                    ? 'Enabled in running config — graph can update from ingest when transports deliver packets.'
                    : 'Disabled — topology pages stay informative but the graph store is not active.'}
              </p>
            </div>
            <div className="rounded-lg border border-border/70 bg-background/80 p-2.5">
              <p className="font-medium text-foreground">Transports</p>
              <p className="mt-1 text-muted-foreground">
                {!anyConfigured && 'None configured — ingest cannot run until transports exist in config.'}
                {anyConfigured &&
                  (anyLive
                    ? `${transports.length} configured; at least one connected/live — evidence can flow.`
                    : `${transports.length} configured; none connected in this poll — evidence may be stale or idle.`)}
              </p>
            </div>
            <div className="rounded-lg border border-border/70 bg-background/80 p-2.5">
              <p className="font-medium text-foreground">Schema / config</p>
              <p className="mt-1 text-muted-foreground">
                Migrations:{' '}
                {schemaOk ? (
                  <span className="text-success">match binary</span>
                ) : (
                  <span className="text-warning">mismatch or unknown — check diagnostics</span>
                )}
                {fp && (
                  <span className="block font-mono text-[10px] mt-0.5 text-muted-foreground/80 truncate" title={fp}>
                    fingerprint {fp.slice(0, 16)}…
                  </span>
                )}
              </p>
            </div>
            <div className="rounded-lg border border-border/70 bg-background/80 p-2.5">
              <p className="font-medium text-foreground">Where to verify</p>
              <ul className="mt-1 space-y-0.5 text-muted-foreground">
                <li>
                  <a href="/status" className="text-primary font-medium hover:underline">
                    Status
                  </a>{' '}
                  — per-transport effective_state and last_ingest
                </li>
                <li>
                  <a href="/diagnostics" className="text-primary font-medium hover:underline">
                    Diagnostics
                  </a>{' '}
                  — internal health signals
                </li>
              </ul>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function ConfigItem({
  name,
  type,
  default: defaultValue,
  description,
}: {
  name: string
  type: string
  default: string
  description: string
}) {
  return (
    <div
      role="group"
      aria-label={`Config key ${name}, read-only reference`}
      className="rounded-lg border border-border p-3"
    >
      <div className="mb-1 flex items-center justify-between gap-2">
        <code className="break-all text-sm font-mono">{name}</code>
        <Badge variant="outline" className="shrink-0 text-xs">
          {type}
        </Badge>
      </div>
      <p className="mb-2 text-xs text-muted-foreground">Documented default: {defaultValue}</p>
      <p className="text-sm">{description}</p>
    </div>
  )
}

function InfoCard({
  title,
  value,
  description,
}: {
  title: string
  value: string
  description: string
}) {
  return (
    <div className="rounded-lg border border-border p-4">
      <p className="text-sm text-muted-foreground">{title}</p>
      <p className="mt-1 break-words text-lg font-semibold">{value}</p>
      <p className="text-xs text-muted-foreground">{description}</p>
    </div>
  )
}

type QuickAccessCardProps =
  | {
      icon: ReactNode
      title: string
      description: string
      href: string
    }
  | {
      icon: ReactNode
      title: string
      description: string
      to: string
    }

function QuickAccessCard(props: QuickAccessCardProps) {
  const { icon, title, description } = props
  const className =
    'flex items-start gap-3 rounded-lg border border-border p-4 transition-colors hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-ring'

  const inner = (
    <>
      <span className="shrink-0 text-muted-foreground" aria-hidden>
        {icon}
      </span>
      <span>
        <span className="font-medium text-foreground">{title}</span>
        <span className="mt-0.5 block text-sm text-muted-foreground">{description}</span>
      </span>
    </>
  )

  if ('to' in props) {
    return (
      <Link to={props.to} className={className}>
        {inner}
      </Link>
    )
  }

  return (
    <a href={props.href} className={className}>
      {inner}
    </a>
  )
}

function DocLink({
  title,
  description,
  href,
  id,
}: {
  title: string
  description: string
  href: string
  id?: string
}) {
  return (
    <a
      id={id}
      href={href}
      className="flex items-start gap-3 rounded-lg border border-border p-4 transition-colors hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <ExternalLink className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" aria-hidden />
      <span>
        <span className="font-medium text-foreground">{title}</span>
        <span className="mt-0.5 block text-sm text-muted-foreground">{description}</span>
      </span>
    </a>
  )
}

function availabilityVariant(a: string): 'success' | 'warning' | 'secondary' | 'outline' {
  if (a === 'available') return 'success'
  if (a === 'partial' || a === 'queued') return 'warning'
  if (a === 'unavailable') return 'secondary'
  return 'outline'
}

function PlatformPostureCard({ posture }: { posture: PlatformPosture }) {
  const ret = posture.retention
  const exp = posture.evidence_export_delete

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Cpu className="h-5 w-5" />
          Effective running posture
        </CardTitle>
        <CardDescription>
          Read from <code className="rounded bg-muted px-1 font-mono text-xs">GET /api/v1/version → platform_posture</code>.
          These are the effective values for the running process — not documented defaults.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Mode + telemetry */}
        <section aria-labelledby="posture-mode-heading">
          <h3 id="posture-mode-heading" className="mb-3 flex items-center gap-2 text-sm font-semibold">
            <Radio className="h-4 w-4 text-muted-foreground" aria-hidden />
            Mode &amp; telemetry
          </h3>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <RunningItem
              name="Platform mode"
              value={posture.mode || '—'}
              source="platform_posture.mode"
              note="Operating mode for this instance"
            />
            <RunningItem
              name="Telemetry enabled"
              value={posture.telemetry_enabled ? 'yes' : 'no'}
              source="platform_posture.telemetry_enabled"
              note="Whether telemetry collection is active"
              variant={posture.telemetry_enabled ? 'warning' : 'success'}
            />
            <RunningItem
              name="Telemetry outbound"
              value={posture.telemetry_outbound ? 'yes' : 'no'}
              source="platform_posture.telemetry_outbound"
              note="Whether telemetry is sent externally"
              variant={posture.telemetry_outbound ? 'warning' : 'success'}
            />
          </div>
        </section>

        {/* Retention */}
        <section aria-labelledby="posture-retention-heading">
          <h3 id="posture-retention-heading" className="mb-3 flex items-center gap-2 text-sm font-semibold">
            <Database className="h-4 w-4 text-muted-foreground" aria-hidden />
            Effective retention
          </h3>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <RunningItem name="Retention active" value={ret.enabled ? 'yes' : 'no'} source="retention.enabled" note="Automatic data pruning on/off" />
            <RunningItem name="Messages" value={`${ret.messages_days}d`} source="retention.messages_days" note="Message retention window" />
            <RunningItem name="Audit logs" value={`${ret.audit_days}d`} source="retention.audit_days" note="Audit log retention window" />
            <RunningItem name="Telemetry" value={`${ret.telemetry_days}d`} source="retention.telemetry_days" note="Telemetry retention window" />
            <RunningItem name="Precise positions" value={`${ret.precise_position_days}d`} source="retention.precise_position_days" note="GPS coordinate retention" />
          </div>
        </section>

        {/* Export / delete */}
        <section aria-labelledby="posture-export-heading">
          <h3 id="posture-export-heading" className="mb-3 flex items-center gap-2 text-sm font-semibold">
            <Shield className="h-4 w-4 text-muted-foreground" aria-hidden />
            Export &amp; delete policy
          </h3>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <RunningItem name="Export enabled" value={exp.export_enabled ? 'yes' : 'no'} source="evidence_export_delete.export_enabled" note="Proofpack / bundle export allowed" />
            <RunningItem name="Delete enabled" value={exp.delete_enabled ? 'yes' : 'no'} source="evidence_export_delete.delete_enabled" note="Operator-triggered data deletion" />
            {exp.delete_scope.length > 0 && (
              <RunningItem name="Delete scope" value={exp.delete_scope.join(', ')} source="evidence_export_delete.delete_scope" note="What can be deleted" />
            )}
            {exp.delete_caveat && (
              <div className="sm:col-span-2 lg:col-span-3 rounded-lg border border-warning/25 bg-warning/5 px-3 py-2 text-xs text-muted-foreground">
                <span className="font-medium text-foreground">Caveat: </span>{exp.delete_caveat}
              </div>
            )}
          </div>
        </section>

        {/* Inference */}
        <section aria-labelledby="posture-inference-heading">
          <h3 id="posture-inference-heading" className="mb-3 flex items-center gap-2 text-sm font-semibold">
            <Cpu className="h-4 w-4 text-muted-foreground" aria-hidden />
            Local inference
          </h3>
          <div className="mb-3">
            <RunningItem
              name="Inference enabled"
              value={posture.inference_enabled ? 'yes' : 'no'}
              source="inference_enabled"
              note="Assistive inference runtime active"
              variant={posture.inference_enabled ? 'success' : 'secondary'}
            />
          </div>
          {posture.inference_providers.length > 0 && (
            <div className="space-y-2">
              <p className="text-xs text-muted-foreground font-medium">Providers</p>
              <div className="grid gap-2 sm:grid-cols-2">
                {posture.inference_providers.map((p) => (
                  <div key={p.name} className="flex items-center justify-between rounded-lg border border-border/60 px-3 py-2 text-xs">
                    <div>
                      <span className="font-medium text-foreground">{p.name}</span>
                      <span className="ml-2 text-muted-foreground">{p.endpoint_configured ? 'endpoint configured' : 'no endpoint'}</span>
                    </div>
                    <Badge variant={p.enabled && p.available_by_config ? 'success' : 'secondary'}>
                      {p.enabled ? (p.available_by_config ? 'available' : 'config-limited') : 'disabled'}
                    </Badge>
                  </div>
                ))}
              </div>
            </div>
          )}
          {posture.assist_policies.length > 0 && (
            <div className="mt-3 space-y-2">
              <p className="text-xs text-muted-foreground font-medium">Assist policies</p>
              <div className="space-y-1.5">
                {posture.assist_policies.map((p) => (
                  <div key={p.task_class} className="flex flex-wrap items-center gap-2 rounded-lg border border-border/50 px-3 py-2 text-xs">
                    <span className="font-mono text-foreground">{p.task_class.replace(/_/g, ' ')}</span>
                    <Badge variant={availabilityVariant(p.availability)}>{p.availability}</Badge>
                    <span className="text-muted-foreground">{p.execution_mode}</span>
                    <span className="text-muted-foreground">{p.provider}</span>
                    <span className="text-muted-foreground">{p.hardware}</span>
                    {p.non_canonical_truth && (
                      <Badge variant="outline" className="text-[10px]">non-canonical</Badge>
                    )}
                    {p.fallback_reason && (
                      <span className="text-warning text-[10px]">{p.fallback_reason}</span>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </section>

        {posture.operator_intelligence_posture && (
          <section aria-labelledby="posture-operator-intel-heading">
            <h3 id="posture-operator-intel-heading" className="mb-3 flex items-center gap-2 text-sm font-semibold">
              <Shield className="h-4 w-4 text-muted-foreground" aria-hidden />
              Operator intelligence posture
            </h3>
            <p className="text-xs text-muted-foreground mb-3">
              Contract for what the UI may claim: deterministic incident intelligence stays on persisted records; assist is optional and never
              canonical truth. Remote cloud assist is not a base product path.
            </p>
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              <RunningItem
                name="Deterministic incident intel"
                value={posture.operator_intelligence_posture.deterministic_incident_intel.replace(/_/g, ' ')}
                source="operator_intelligence_posture.deterministic_incident_intel"
                note={posture.operator_intelligence_posture.deterministic_basis.replace(/_/g, ' ')}
              />
              <RunningItem
                name="Assistive inference layer"
                value={posture.operator_intelligence_posture.assistive_inference_layer.replace(/_/g, ' ')}
                source="operator_intelligence_posture.assistive_inference_layer"
                note="LLM / assist when enabled — review output; does not override ingest or audit truth"
                variant={posture.operator_intelligence_posture.assistive_inference_layer === 'available' ? 'warning' : 'secondary'}
              />
              {posture.operator_intelligence_posture.assist_capability_strategy && (
                <RunningItem
                  name="Assist capability strategy"
                  value={posture.operator_intelligence_posture.assist_capability_strategy.replace(/_/g, ' ')}
                  source="operator_intelligence_posture.assist_capability_strategy"
                  note="Explicit contract for future assist surfaces — local-first; no hidden cloud path"
                  variant={
                    posture.operator_intelligence_posture.assist_capability_strategy === 'enabled_bounded_local_assist'
                      ? 'warning'
                      : posture.operator_intelligence_posture.assist_capability_strategy === 'unavailable' ||
                          posture.operator_intelligence_posture.assist_capability_strategy === 'not_configured'
                        ? 'outline'
                        : 'secondary'
                  }
                />
              )}
              <RunningItem
                name="Remote assist supported"
                value={posture.operator_intelligence_posture.remote_assist_supported ? 'yes' : 'no'}
                source="operator_intelligence_posture.remote_assist_supported"
                note="Base MEL is local-first; no mandatory cloud assist"
                variant={posture.operator_intelligence_posture.remote_assist_supported ? 'warning' : 'success'}
              />
              <RunningItem
                name="Telemetry outbound"
                value={posture.operator_intelligence_posture.telemetry_outbound ? 'yes' : 'no'}
                source="operator_intelligence_posture.telemetry_outbound"
                note="Outbound telemetry requires explicit opt-in in validated configs"
                variant={posture.operator_intelligence_posture.telemetry_outbound ? 'warning' : 'success'}
              />
            </div>
            {(posture.operator_intelligence_posture.assist_input_contracts?.length ?? 0) > 0 && (
              <div className="mt-4 rounded-lg border border-border/60 bg-muted/20 p-3 text-xs text-muted-foreground">
                <p className="font-semibold text-foreground mb-1">Future bounded assist — canonical inputs (design contract)</p>
                <ul className="list-disc pl-4 space-y-0.5 font-mono text-[10px]">
                  {posture.operator_intelligence_posture.assist_input_contracts!.map((c) => (
                    <li key={c}>{c}</li>
                  ))}
                </ul>
                {posture.operator_intelligence_posture.assist_disable_semantics && (
                  <p className="mt-2">{posture.operator_intelligence_posture.assist_disable_semantics}</p>
                )}
                {posture.operator_intelligence_posture.assist_audit_expectation && (
                  <p className="mt-2 border-t border-border/40 pt-2">{posture.operator_intelligence_posture.assist_audit_expectation}</p>
                )}
              </div>
            )}
          </section>
        )}
      </CardContent>
    </Card>
  )
}

function RunningItem({
  name,
  value,
  source,
  note,
  variant,
}: {
  name: string
  value: string
  source: string
  note: string
  variant?: 'success' | 'warning' | 'secondary' | 'outline'
}) {
  return (
    <div className="rounded-lg border border-border p-3">
      <div className="mb-1 flex items-center justify-between gap-2">
        <span className="text-sm font-medium text-foreground">{name}</span>
        {variant && <Badge variant={variant} className="shrink-0 text-xs">{value}</Badge>}
      </div>
      {!variant && <p className="text-base font-semibold text-foreground">{value}</p>}
      <p className="mt-1 text-xs text-muted-foreground">{note}</p>
      <p className="mt-0.5 font-mono text-[10px] text-muted-foreground/60">{source}</p>
    </div>
  )
}
