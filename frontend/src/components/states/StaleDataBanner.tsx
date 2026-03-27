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
    <div
      className="mb-4 rounded-md border border-warning/25 border-l-4 border-l-warning bg-warning/10 p-4 shadow-sm"
      role="alert"
    >
      <div className="flex">
        <div className="min-w-0">
          <h3 className="text-sm font-medium text-foreground">
            Stale data — {componentName}
          </h3>
          <div className="mt-1 text-sm text-muted-foreground">
            <p>
              No recent updates received. Last verified activity:{' '}
              <strong className="font-medium text-foreground">{formatOperatorTime(lastSuccessfulIngest)}</strong>.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};