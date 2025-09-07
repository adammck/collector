import { useAppStore } from '../store';

export function QueueStatus() {
  const { queue } = useAppStore();
  
  if (!queue) return null;
  
  const { total, active, deferred } = queue;
  
  let statusText = `Queue: ${active} active`;
  if (deferred > 0) statusText += `, ${deferred} deferred`;
  statusText += `, ${total} total`;
  
  return (
    <div className="flex items-center gap-2">
      <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
      <span className="px-3 py-2 bg-gray-100 border border-gray-300 rounded-lg text-sm font-medium text-gray-700">
        {statusText}
      </span>
    </div>
  );
}