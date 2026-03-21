import { useEffect, useState } from 'react';
import { SupportBundleExport } from '@/components/diagnostics/SupportBundleExport';
import { OperatorEmptyState } from '@/components/states/OperatorEmptyState';
import { safeArray } from '@/utils/apiResilience';
import { PageHeader } from '@/components/ui/PageHeader';

interface DiagnosticFinding {
  code: string;
  severity: 'critical' | 'warning' | 'info';
  component: string;
  title: string;
  explanation: string;
  recommended_steps: string[];
}

type DiagnosticsPageState = 'loading' | 'unreachable' | 'disabled' | 'ready';

export function Diagnostics() {
  const [findings, setFindings] = useState<DiagnosticFinding[]>([]);
  const [pageState, setPageState] = useState<DiagnosticsPageState>('loading');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchDiagnostics = async () => {
      try {
        const res = await fetch('/api/v1/diagnostics');
        if (res.status === 404 || res.status === 501) {
           setPageState('disabled');
           return;
        }
        if (!res.ok) throw new Error(`HTTP ${res.status}: Failed to fetch diagnostics`);
        const data = await res.json();
        setFindings(safeArray(data.findings));
        setPageState('ready');
      } catch (err) {
        setPageState('unreachable');
        setError(err instanceof TypeError ? 'Backend is unreachable (Network Error). Is MEL running?' : (err as Error).message);
      }
    };
    fetchDiagnostics();
  }, []);

  if (pageState === 'loading') {
    return <div className="p-8 text-gray-500 animate-pulse">Running system diagnostics...</div>;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="System Diagnostics"
        description="Real-time health evaluation of transports, databases, and control limits."
      />

      <div className="mb-8">
        <SupportBundleExport />
      </div>

      {pageState === 'unreachable' && (
        <div className="bg-red-50 p-4 rounded-md mb-6 border border-red-200 text-red-700">
          <strong>Connection Failure:</strong> {error}
        </div>
      )}

      {pageState === 'disabled' && (
        <div className="bg-gray-50 p-4 rounded-md mb-6 border border-gray-200 text-gray-700">
          <strong>Diagnostics Unavailable:</strong> The MEL backend running does not yet support the `/api/v1/diagnostics` API. Use <code className="text-xs">mel doctor</code> via CLI.
        </div>
      )}

      <div className="space-y-4">
        {pageState === 'ready' && findings.length === 0 ? (
          <OperatorEmptyState 
            title="No diagnostic findings" 
            description="MEL reports 0 active diagnostic findings. System checks passed." 
          />
        ) : pageState === 'ready' && (
          findings.map((f, idx) => (
            <div key={idx} className={`p-5 rounded-lg border ${f.severity === 'critical' ? 'bg-red-50 border-red-200' : f.severity === 'warning' ? 'bg-yellow-50 border-yellow-200' : 'bg-blue-50 border-blue-200'}`}>
              <div className="flex justify-between items-start mb-2">
                <h4 className={`text-lg font-semibold ${f.severity === 'critical' ? 'text-red-800' : f.severity === 'warning' ? 'text-yellow-800' : 'text-blue-800'}`}>
                  {f.title}
                </h4>
                <span className="px-2 py-1 text-xs rounded-full bg-white bg-opacity-50 text-gray-700 border shadow-sm uppercase font-mono">
                  {f.code}
                </span>
              </div>
              <p className="text-sm text-gray-700 mb-4">{f.explanation}</p>
              <div className="text-sm font-medium text-gray-900 mb-1">Recommended Action:</div>
              <ul className="list-disc pl-5 space-y-1 text-sm text-gray-700">
                {safeArray(f.recommended_steps).map((step, sIdx) => <li key={sIdx}>{step}</li>)}
              </ul>
            </div>
          ))
        )}
      </div>
    </div>
  );
}