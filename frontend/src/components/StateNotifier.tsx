import { useAppStore } from '../store';

const stateMessages = {
  idle: { className: 'text-blue-600 bg-blue-50 border-blue-200', icon: '⏸️', text: 'Idle' },
  awaiting_data: { className: 'text-amber-600 bg-amber-50 border-amber-200', icon: '⏳', text: 'Waiting for data...' },
  waiting_user: { className: 'text-green-600 bg-green-50 border-green-200', icon: '✅', text: 'Ready for response' },
  submitting: { className: 'text-blue-600 bg-blue-50 border-blue-200', icon: '📤', text: 'Submitting...' },
  server_error: { className: 'text-red-600 bg-red-50 border-red-200', icon: '⚠️', text: 'Server error' },
  client_error: { className: 'text-red-600 bg-red-50 border-red-200', icon: '⚠️', text: 'Client error' },
};

export function StateNotifier() {
  const { state, stateMessage } = useAppStore();
  
  const config = stateMessages[state] || { className: 'text-gray-600', text: 'Unknown state' };
  const displayText = stateMessage || config.text;
  
  return (
    <div className={`px-3 py-2 text-sm font-medium rounded-lg border flex items-center gap-2 ${config.className}`}>
      <span>{config.icon}</span>
      <span>{displayText}</span>
    </div>
  );
}