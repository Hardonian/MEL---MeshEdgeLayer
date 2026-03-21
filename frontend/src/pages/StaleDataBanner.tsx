import React from 'react';
import { evaluateTimeState, formatOperatorTime } from '@/utils/apiResilience';

interface StaleDataBannerProps {
  lastSuccessfulIngest: string | null | undefined;
  thresholdMinutes?: number;
  componentName: string;
}

export const StaleDataBanner: React.FC<StaleDataBannerProps> = ({ 
  lastSuccessfulIngest, 
  thresholdMinutes = 5,
  componentName 
}) => {
  const thresholdMs = thresholdMinutes * 60 * 1000;
  const state = evaluateTimeState(lastSuccessfulIngest, thresholdMs);

  if (state !== 'stale') return null;

  return (
    <div className="bg-amber-50 border-l-4 border-amber-400 p-4 mb-4 rounded-md shadow-sm" role="alert">
      <div className="flex">
        <div className="ml-3">
          <h3 className="text-sm font-medium text-amber-800">Stale Data Warning: {componentName}</h3>
          <div className="mt-1 text-sm text-amber-700">
            <p>No recent updates received. Last verified activity: <strong>{formatOperatorTime(lastSuccessfulIngest)}</strong>.</p>
          </div>
        </div>
      </div>
    </div>
  );
};