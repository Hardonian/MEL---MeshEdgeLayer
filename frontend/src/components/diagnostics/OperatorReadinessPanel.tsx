import { useEffect, useState } from 'react';

type ReadyReasonCode = string;

interface ReadyComponent {
  name: string;
  state: string;
  detail?: string;
}

interface ReadyzPayload {
  api_version?: string;
  ready?: boolean;
  status?: string;
  reason_codes?: ReadyReasonCode[];
  summary?: string;
  checked_at?: string;
  process_ready?: boolean;
  ingest_ready?: boolean;
  stale_ingest_evidence?: boolean;
  operator_state?: string;
  mesh_state?: string;
  operator_next_steps?: string[];
  components?: ReadyComponent[];
  error_class?: string;
  message?: string;
}

interface StatusPayload {
  status?: {
    transports?: Array<{ name?: string; effective_state?: string; detail?: string }>;
    last_successful_ingest?: string;
  };
}

function pillClass(ok: boolean | undefined, warn?: boolean) {
  if (ok === true) return 'bg-emerald-50 text-emerald-900 border-emerald-200';
  if (warn) return 'bg-amber-50 text-amber-900 border-amber-200';
  return 'bg-rose-50 text-rose-900 border-rose-200';
}

export const OperatorReadinessPanel: React.FC = () => {
  const [phase, setPhase] = useState<'loading' | 'ok' | 'error'>('loading');
  const [err, setErr] = useState<string | null>(null);
  const [readyz, setReadyz] = useState<ReadyzPayload | null>(null);
  const [statusSnap, setStatusSnap] = useState<StatusPayload | null>(null);

  useEffect(() => {
    const run = async () => {
      try {
        const [rRes, sRes] = await Promise.all([
          fetch('/api/v1/readyz'),
          fetch('/api/v1/status'),
        ]);
        const rJson = (await rRes.json()) as ReadyzPayload;
        setReadyz(rJson);
        if (sRes.ok) {
          setStatusSnap((await sRes.json()) as StatusPayload);
        }
        setPhase('ok');
      } catch (e) {
        setPhase('error');
        setErr(e instanceof TypeError ? 'API unreachable — is MEL running and is this page same-origin?' : String(e));
      }
    };
    void run();
  }, []);

  if (phase === 'loading') {
    return (
      <div className="p-6 border border-gray-200 rounded-lg bg-white text-gray-500 text-sm animate-pulse">
        Loading readiness and status from the API…
      </div>
    );
  }

  if (phase === 'error') {
    return (
      <div className="p-6 border border-red-200 rounded-lg bg-red-50 text-red-800 text-sm">
        <strong>Diagnostics API unreachable.</strong> {err}
        <p className="mt-2 text-gray-700">
          Quick process liveness only: <code className="text-xs">GET /healthz</code> (or <code className="text-xs">mel preflight</code>).
          Readiness needs a running API: <code className="text-xs">GET /api/v1/readyz</code>.
        </p>
      </div>
    );
  }

  const httpReady = readyz?.ready === true;
  const idle =
    readyz?.operator_state === 'idle' ||
    (readyz?.reason_codes?.includes('TRANSPORT_IDLE') ?? false);
  const degraded = readyz?.operator_state === 'degraded';

  return (
    <div className="space-y-4 p-6 border border-gray-200 rounded-lg bg-white shadow-sm">
      <div>
        <h3 className="text-lg font-semibold text-gray-900">Setup &amp; readiness</h3>
        <p className="text-sm text-gray-600 mt-1">
          Signals come from <code className="text-xs">/api/v1/readyz</code> and <code className="text-xs">/api/v1/status</code> — not invented UI state.
        </p>
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        <div className={`rounded-md border p-3 text-sm ${pillClass(true)}`}>
          <div className="font-medium">Liveness (process up)</div>
          <p className="mt-1 opacity-90">
            <code className="text-xs">GET /healthz</code> — JSON <code className="text-xs">{"{ ok: true }"}</code> only proves the HTTP process answers.
            Preflight uses this for a quick probe; it does not prove MQTT, serial, or ingest.
          </p>
        </div>
        <div className={`rounded-md border p-3 text-sm ${pillClass(httpReady, idle && httpReady)}`}>
          <div className="font-medium">Readiness (subsystem + ingest contract)</div>
          <p className="mt-1 opacity-90">
            <code className="text-xs">GET /readyz</code> and <code className="text-xs">GET /api/v1/readyz</code> share semantics: HTTP <strong>200</strong> when ready,{' '}
            <strong>503</strong> when not (or when the status snapshot cannot be built). Idle (no enabled transports) returns 200 with explicit idle reasons.
          </p>
        </div>
      </div>

      <div className="rounded-md border border-gray-100 bg-gray-50 p-4 text-sm text-gray-800">
        <div className="flex flex-wrap gap-2 items-center">
          <span className={`px-2 py-0.5 rounded text-xs font-mono border ${pillClass(httpReady, idle)}`}>
            ready={String(readyz?.ready)} · status={readyz?.status ?? '—'}
          </span>
          {readyz?.ingest_ready === false && !idle && (
            <span className="text-xs text-amber-800">Ingest not proven on any transport</span>
          )}
          {idle && <span className="text-xs text-gray-700">Transport idle (no enabled transports)</span>}
          {degraded && <span className="text-xs text-amber-800">Degraded transport evidence — see status</span>}
          {readyz?.stale_ingest_evidence && (
            <span className="text-xs text-amber-800">Stale persisted ingest timestamp</span>
          )}
        </div>
        {readyz?.summary && <p className="mt-2">{readyz.summary}</p>}
        {readyz?.reason_codes && readyz.reason_codes.length > 0 && (
          <p className="mt-2 text-xs font-mono text-gray-600">reason_codes: {readyz.reason_codes.join(', ')}</p>
        )}
        {readyz?.error_class === 'snapshot_unavailable' && (
          <p className="mt-2 text-rose-800">
            Readiness evidence unavailable (503). Process is up; fix DB/migrations or paths, then retry.
          </p>
        )}
      </div>

      {statusSnap?.status?.transports && statusSnap.status.transports.length > 0 && (
        <div className="text-sm">
          <div className="font-medium text-gray-900 mb-2">Transport truth (from /api/v1/status)</div>
          <ul className="space-y-1 text-gray-700">
            {statusSnap.status.transports.slice(0, 8).map((tr, i) => (
              <li key={i} className="flex flex-wrap gap-2">
                <span className="font-mono text-xs">{tr.name}</span>
                <span className="text-xs bg-gray-100 px-1 rounded">{tr.effective_state}</span>
                {tr.detail && <span className="text-xs text-gray-600">{tr.detail}</span>}
              </li>
            ))}
          </ul>
        </div>
      )}

      {readyz?.components && readyz.components.length > 0 && (
        <div className="text-sm">
          <div className="font-medium text-gray-900 mb-2">Components</div>
          <ul className="grid gap-1 sm:grid-cols-2">
            {readyz.components.map((c, i) => (
              <li key={i} className="text-xs border rounded px-2 py-1 bg-white">
                <span className="font-mono">{c.name}</span>: {c.state}
                {c.detail && <span className="text-gray-600"> — {c.detail}</span>}
              </li>
            ))}
          </ul>
        </div>
      )}

      {readyz?.operator_next_steps && readyz.operator_next_steps.length > 0 && (
        <div>
          <div className="font-medium text-sm text-gray-900 mb-1">Suggested next steps</div>
          <ol className="list-decimal pl-5 space-y-1 text-sm text-gray-700">
            {readyz.operator_next_steps.map((s, i) => (
              <li key={i}>{s}</li>
            ))}
          </ol>
        </div>
      )}

      <div className="text-xs text-gray-600 border-t pt-3 space-y-1">
        <p>
          <strong>Deeper diagnosis:</strong> run <code>mel doctor</code> or <code>mel preflight</code> on the host (paths, DB, serial/TCP checks).
        </p>
        <p>
          <strong>Authoritative transport view:</strong> <code>GET /api/v1/status</code> and <code>mel status --config …</code>.
        </p>
      </div>
    </div>
  );
};
