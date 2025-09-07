import { useAppStore } from '../store';

const stateMessages = {
  idle: { className: 'text-blue-600', text: 'Idle.' },
  awaiting_data: { className: 'text-blue-600', text: 'Waiting for data...' },
  waiting_user: { className: 'text-blue-600', text: 'Waiting for user...' },
  submitting: { className: 'text-blue-600', text: 'Submitting response...' },
  server_error: { className: 'text-red-600', text: 'Server error' },
  client_error: { className: 'text-red-600', text: 'Application error' },
};

export function StateNotifier() {
  const { state, stateMessage } = useAppStore();
  
  const config = stateMessages[state] || { className: 'text-gray-600', text: 'Unknown state' };
  const displayText = stateMessage || config.text;
  
  return (
    <div className={`px-2 py-1 text-sm ${config.className}`}>
      {displayText}
    </div>
  );
}