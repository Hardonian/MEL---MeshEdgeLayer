import React, { useState } from 'react';

export const SupportBundleExport: React.FC = () => {
  const [status, setStatus] = useState<'idle' | 'generating' | 'success' | 'error' | 'unavailable'>('idle');
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const handleExport = async () => {
    setStatus('generating');
    setErrorMessage(null);
    
    try {
      const response = await fetch('/api/v1/support-bundle');
      if (response.status === 404 || response.status === 501) {
        setStatus('unavailable');
        setErrorMessage('Support bundle export API is currently disabled or not implemented in this backend version.');
        return;
      }
      if (response.status === 401 || response.status === 403) {
        throw new Error(`${response.status}: authentication or capability required (export_support_bundle). Use mel support bundle --config … from the host if the UI cannot authorize.`);
      }
      if (!response.ok) {
        throw new Error(`Export failed: ${response.status} ${response.statusText}`);
      }
      
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `mel-support-bundle-${new Date().toISOString().replace(/[:.]/g, '-')}.zip`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      
      setStatus('success');
      setTimeout(() => setStatus('idle'), 5000);
    } catch (err) {
      setStatus('error');
      setErrorMessage(
        err instanceof TypeError ? 'MEL backend unreachable (Network Error).' : 
        err instanceof Error ? err.message : 'Unknown network error'
      );
    }
  };

  return (
    <div className="bg-white p-6 border border-gray-200 rounded-lg shadow-sm">
      <h3 className="text-lg font-medium text-gray-900 mb-2">Export Support Bundle</h3>
      <p className="text-sm text-gray-600 mb-4">
        Downloads a ZIP with <code className="text-xs">bundle.json</code> (redacted config, nodes, messages sample, control evidence) and{' '}
        <code className="text-xs">doctor.json</code> (same checks as <code className="text-xs">mel doctor</code>, bundle-redacted). Review before sharing externally.
      </p>
      <button
        onClick={handleExport}
        disabled={status === 'generating' || status === 'unavailable'}
        className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {status === 'generating' ? 'Generating Bundle...' : 'Download Redacted Bundle'}
      </button>
      {(status === 'error' || status === 'unavailable') && (
        <p className="mt-2 text-sm text-red-600">Failed to generate bundle: {errorMessage}</p>
      )}
    </div>
  );
};