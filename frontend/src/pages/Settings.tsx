import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { KeyboardShortcuts } from '@/components/ui/HelpMenu'
import { useConsoleThemePreference } from '@/hooks/useConsoleThemePreference'
import { useVersionInfo } from '@/hooks/useVersionInfo'
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
} from 'lucide-react'

export function SettingsPage() {
  const version = useVersionInfo()
  const { preference, setPreference } = useConsoleThemePreference()

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

      {/* Note about config editing */}
      <AlertCard
        variant="info"
        title="Configuration is file-based"
        description="MEL reads settings from a JSON config file on the host. Edit that file and restart the process for changes to apply. This UI does not persist server configuration."
      />
    </div>
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
