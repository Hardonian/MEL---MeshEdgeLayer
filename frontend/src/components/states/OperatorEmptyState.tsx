import React from 'react';

interface OperatorEmptyStateProps {
  title: string;
  description: string;
  actionNode?: React.ReactNode;
}

export const OperatorEmptyState: React.FC<OperatorEmptyStateProps> = ({ title, description, actionNode }) => {
  return (
    <div className="flex flex-col items-center justify-center p-12 bg-gray-50 border border-dashed border-gray-300 rounded-lg">
      <svg className="w-12 h-12 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
      </svg>
      <h3 className="text-sm font-semibold text-gray-900">{title}</h3>
      <p className="mt-1 text-sm text-gray-500 max-w-sm text-center">{description}</p>
      
      {actionNode && (
        <div className="mt-6">
          {actionNode}
        </div>
      )}
    </div>
  );
};