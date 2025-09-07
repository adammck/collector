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
    <div className="h-screen flex flex-col">
      <div className="flex flex-1">
        <div className="w-1/2 bg-gray-100">
          {input && <GridVisualization input={input} />}
        </div>
        <div className="w-1/2 bg-white flex justify-center items-center">
          {output && (
            <OptionList
              output={output}
              disabled={isSubmitting}
              onSubmit={handleSubmit}
            />
          )}
        </div>
      </div>
      
      <div className="h-25 bg-gray-200 flex items-center justify-between px-5">
        <QueueStatus />
        <StateNotifier />
        <div className="flex gap-2">
          <button
            className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
            onClick={handleNext}
          >
            Fetch Data
          </button>
          <button
            className="px-4 py-2 bg-orange-500 text-white rounded hover:bg-orange-600"
            onClick={handleDefer}
            disabled={!currentUUID}
            title="Ctrl+D"
          >
            Defer
          </button>
        </div>
      </div>
    </div>
  );
}