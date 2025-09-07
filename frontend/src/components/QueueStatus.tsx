import { useAppStore } from '../store';

export function QueueStatus() {
  const { queue } = useAppStore();
  
  if (!queue) return null;
  
  const { total, active, deferred } = queue;
  
  let statusText = `Queue: ${active} active`;
  if (deferred > 0) statusText += `, ${deferred} deferred`;
  statusText += `, ${total} total`;
  
  return (
    <div className="text-sm text-gray-700">
      <span className="px-2 py-1 bg-white border border-gray-300 rounded">
        {statusText}
      </span>
    </div>
  );
}