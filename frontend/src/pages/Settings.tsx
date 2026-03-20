import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { Server, Info, ExternalLink, Lock, Database, Wifi, Shield, Terminal, BookOpen, Wrench } from 'lucide-react'

export function SettingsPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="Settings"
        description="MEL configuration reference, system information, and documentation links."
      />

      {/* Quick Access */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <QuickAccessCard
          icon={<Wrench className="h-5 w-5" />}
          title="Configuration"
          description="Edit your MEL config file"
          href="#config"
        />
        <QuickAccessCard
          icon={<Database className="h-5 w-5" />}
          title="Storage"
          description="Data directory & retention"
          href="#storage"
        />
        <QuickAccessCard
          icon={<Shield className="h-5 w-5" />}
          title="Privacy"
          description="Privacy settings"
          href="#privacy"
        />
        <QuickAccessCard
          icon={<Terminal className="h-5 w-5" />}
          title="CLI"
          description="Command-line reference"
          href="#cli"
        />
      </div>

      {/* Configuration Reference */}
      <Card id="config">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Server className="h-5 w-5" />
            Configuration Reference
          </CardTitle>
          <CardDescription>
            MEL configuration options and their defaults
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-8">
            {/* Network */}
            <section id="network">
              <h3 className="text-sm font-semibold mb-3 flex items-center gap-2">
                <Wifi className="h-4 w-4 text-muted-foreground" />
                Network Binding
              </h3>
              <div className="grid gap-3 sm:grid-cols-2">
                <ConfigItem name="bind.api" type="string" default='"127.0.0.1:8080"' description="HTTP API listen address" />
                <ConfigItem name="bind.metrics" type="string" default='""' description="Prometheus metrics endpoint" />
              </div>
            </section>

            {/* Auth */}
            <section id="auth">
              <h3 className="text-sm font-semibold mb-3 flex items-center gap-2">
                <Lock className="h-4 w-4 text-muted-foreground" />
                Authentication
              </h3>
              <div className="grid gap-3 sm:grid-cols-2">
                <ConfigItem name="auth.enabled" type="bool" default="false" description="Enable basic auth for UI" />
                <ConfigItem name="auth.ui_user" type="string" default='"admin"' description="Username for UI access" />
              </div>
            </section>

            {/* Storage */}
            <section id="storage">
              <h3 className="text-sm font-semibold mb-3 flex items-center gap-2">
                <Database className="h-4 w-4 text-muted-foreground" />
                Storage
              </h3>
              <div className="grid gap-3 sm:grid-cols-2">
                <ConfigItem name="storage.data_dir" type="string" default='"./data"' description="Directory for MEL data" />
                <ConfigItem name="storage.database_path" type="string" default='"./data/mel.db"' description="SQLite database path" />
                <ConfigItem name="retention.messages_days" type="int" default="30" description="Message retention period" />
                <ConfigItem name="retention.audit_days" type="int" default="90" description="Audit log retention" />
              </div>
            </section>

            {/* Privacy */}
            <section id="privacy">
              <h3 className="text-sm font-semibold mb-3 flex items-center gap-2">
                <Shield className="h-4 w-4 text-muted-foreground" />
                Privacy
              </h3>
              <div className="grid gap-3 sm:grid-cols-2">
                <ConfigItem name="privacy.store_precise_positions" type="bool" default="false" description="Store exact GPS coordinates" />
                <ConfigItem name="privacy.mqtt_encryption_required" type="bool" default="true" description="Require TLS for MQTT" />
                <ConfigItem name="privacy.map_reporting_allowed" type="bool" default="false" description="Allow map reporting" />
                <ConfigItem name="privacy.redact_exports" type="bool" default="true" description="Redact sensitive data in exports" />
              </div>
            </section>

            {/* Features */}
            <section id="features">
              <h3 className="text-sm font-semibold mb-3">Features</h3>
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
            System Information
          </CardTitle>
          <CardDescription>
            MEL version and runtime details
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <InfoCard
              title="Version"
              value="1.0.0"
              description="MEL semantic version"
            />
            <InfoCard
              title="Go Version"
              value="1.25+"
              description="Runtime environment"
            />
            <InfoCard
              title="API Version"
              value="v1"
              description="REST API version"
            />
            <InfoCard
              title="Frontend"
              value="React"
              description="Web UI framework"
            />
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
            Additional resources for configuring and using MEL
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2">
            <DocLink
              title="Configuration Guide"
              description="Full configuration reference with all options explained"
              href="/docs/ops/configuration.md"
            />
            <DocLink
              title="CLI Reference"
              description="Command-line interface documentation"
              href="/docs/ops/cli-reference.md"
            />
            <DocLink
              title="First 10 Minutes"
              description="Quick start guide to get MEL running"
              href="/docs/ops/first-10-minutes.md"
            />
            <DocLink
              title="API Reference"
              description="REST API endpoints and schemas"
              href="/docs/ops/api-reference.md"
            />
            <DocLink
              title="Transport Matrix"
              description="Supported transport protocols and their capabilities"
              href="/docs/ops/transport-matrix.md"
            />
            <DocLink
              title="Troubleshooting"
              description="Common issues and how to resolve them"
              href="/docs/ops/troubleshooting.md"
            />
          </div>
        </CardContent>
      </Card>

      {/* Note about config editing */}
      <AlertCard
        variant="info"
        title="Configuration is file-based"
        description="MEL stores configuration in a JSON file. Edit the config file directly and restart MEL for changes to take effect. The UI provides a read-only view of current settings."
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
    <div className="rounded-lg border p-3 hover:bg-muted/30 transition-colors">
      <div className="flex items-center justify-between mb-1">
        <code className="text-sm font-mono">{name}</code>
        <Badge variant="outline" className="text-xs">{type}</Badge>
      </div>
      <p className="text-xs text-muted-foreground mb-2">default: {defaultValue}</p>
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
    <div className="rounded-lg border p-4">
      <p className="text-sm text-muted-foreground">{title}</p>
      <p className="text-lg font-semibold mt-1">{value}</p>
      <p className="text-xs text-muted-foreground">{description}</p>
    </div>
  )
}

function QuickAccessCard({
  icon,
  title,
  description,
  href,
}: {
  icon: React.ReactNode
  title: string
  description: string
  href: string
}) {
  return (
    <a
      href={href}
      className="flex items-start gap-3 rounded-lg border p-4 hover:bg-accent transition-colors"
    >
      <div className="shrink-0 text-muted-foreground">
        {icon}
      </div>
      <div>
        <p className="font-medium">{title}</p>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
    </a>
  )
}

function DocLink({
  title,
  description,
  href,
}: {
  title: string
  description: string
  href: string
}) {
  return (
    <a
      href={href}
      className="flex items-start gap-3 rounded-lg border p-4 hover:bg-accent transition-colors"
    >
      <div className="shrink-0 text-muted-foreground">
        <ExternalLink className="h-4 w-4" />
      </div>
      <div>
        <p className="font-medium">{title}</p>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
    </a>
  )
}
