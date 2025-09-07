import { useQuery, useMutation } from '@tanstack/react-query';
import { useEffect } from 'react';
import { GridVisualization } from './components/GridVisualization';
import { OptionList } from './components/OptionList';
import { QueueStatus } from './components/QueueStatus';
import { StateNotifier } from './components/StateNotifier';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { useAppStore } from './store';
import { fetchData, submitResponse, deferItem, APIError } from './api';

export function CollectorApp() {
  const { currentUUID, state, setCurrentUUID, setState, setQueue } = useAppStore();

  const dataQuery = useQuery({
    queryKey: ['data'],
    queryFn: fetchData,
    refetchInterval: state === 'awaiting_data' ? 1000 : false,
  });

  useEffect(() => {
    if (dataQuery.data) {
      setCurrentUUID(dataQuery.data.uuid);
      setQueue(dataQuery.data.queue);
      setState('waiting_user');
    }
  }, [dataQuery.data, setCurrentUUID, setQueue, setState]);

  useEffect(() => {
    if (dataQuery.error) {
      const error = dataQuery.error as APIError;
      if (error.message.includes('timeout')) {
        setState('server_error', 'no training data available - waiting...');
      } else if (error.name === 'TypeError' || error.message.includes('network')) {
        setState('client_error', 'network error - check connection');
      } else {
        setState('client_error', error.message);
      }
    }
  }, [dataQuery.error, setState]);

  const submitMutation = useMutation({
    mutationFn: ({ uuid, index }: { uuid: string; index: number }) =>
      submitResponse(uuid, index),
    onMutate: () => {
      setState('submitting');
    },
    onSuccess: () => {
      dataQuery.refetch();
    },
    onError: (error: APIError) => {
      if (error.message.includes('timeout')) {
        setState('server_error', 'no training data available - waiting...');
      } else if (error.name === 'TypeError' || error.message.includes('network')) {
        setState('client_error', 'network error - check connection');
      } else {
        setState('client_error', error.message);
      }
      
      setTimeout(() => dataQuery.refetch(), 5000);
    },
  });

  const deferMutation = useMutation({
    mutationFn: (uuid: string) => deferItem(uuid),
    onMutate: () => {
      setState('awaiting_data', 'deferring item...');
    },
    onSuccess: (data) => {
      setCurrentUUID(data.uuid);
      setQueue(data.queue);
      setState('waiting_user');
    },
    onError: (error: APIError) => {
      setState('client_error', error.message);
      setTimeout(() => dataQuery.refetch(), 5000);
    },
  });

  const handleSubmit = (index: number) => {
    if (!currentUUID || state !== 'waiting_user') return;
    submitMutation.mutate({ uuid: currentUUID, index });
  };

  const handleDefer = () => {
    if (!currentUUID) return;
    deferMutation.mutate(currentUUID);
  };

  const handleNext = () => {
    setState('awaiting_data');
    dataQuery.refetch();
  };

  useKeyboardShortcuts({
    onDefer: handleDefer,
    onNext: handleNext,
  });

  const data = dataQuery.data;
  const input = data?.proto.inputs?.[0];
  const output = data?.proto.output?.Output;
  const isSubmitting = state === 'submitting';

  return (
    <div className="h-screen flex flex-col bg-gradient-to-br from-slate-50 to-gray-100">
      {/* Header */}
      <div className="bg-white shadow-md border-b border-gray-200">
        <div className="px-6 py-4">
          <h1 className="text-2xl font-bold text-gray-800">Collector</h1>
          <p className="text-sm text-gray-600">Human-in-the-loop data collection interface</p>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex flex-1 gap-6 p-6">
        {/* Left Panel - Grid Visualization */}
        <div className="flex-1 bg-white rounded-xl shadow-lg border border-gray-200 overflow-hidden">
          <div className="bg-gray-50 px-6 py-4 border-b border-gray-200">
            <h2 className="text-lg font-semibold text-gray-800">Input Data</h2>
            <p className="text-sm text-gray-600">Grid visualization of the current data sample</p>
          </div>
          <div className="p-6">
            {input ? (
              <GridVisualization input={input} />
            ) : (
              <div className="flex items-center justify-center h-64 text-gray-500">
                <div className="text-center">
                  <div className="text-4xl mb-2">📊</div>
                  <div>No data available</div>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Right Panel - Options */}
        <div className="w-96 bg-white rounded-xl shadow-lg border border-gray-200 overflow-hidden">
          <div className="bg-gray-50 px-6 py-4 border-b border-gray-200">
            <h2 className="text-lg font-semibold text-gray-800">Response Options</h2>
            <p className="text-sm text-gray-600">Select your response or use keyboard shortcuts</p>
          </div>
          <div className="p-6">
            {output ? (
              <OptionList
                output={output}
                disabled={isSubmitting}
                onSubmit={handleSubmit}
              />
            ) : (
              <div className="flex items-center justify-center h-64 text-gray-500">
                <div className="text-center">
                  <div className="text-4xl mb-2">⏳</div>
                  <div>Waiting for options</div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
      
      {/* Footer */}
      <div className="bg-white shadow-md border-t border-gray-200">
        <div className="px-6 py-4 flex items-center justify-between">
          <QueueStatus />
          <StateNotifier />
          <div className="flex gap-3">
            <button
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors shadow-sm font-medium"
              onClick={handleNext}
            >
              Fetch Data
            </button>
            <button
              className="px-4 py-2 bg-amber-600 text-white rounded-lg hover:bg-amber-700 transition-colors shadow-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed"
              onClick={handleDefer}
              disabled={!currentUUID}
              title="Ctrl+D"
            >
              Defer
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}