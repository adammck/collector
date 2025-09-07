import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CollectorApp } from './CollectorApp';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error: any) => {
        // retry on timeout, unavailable, resource exhausted
        if (error?.status === 408 || error?.status === 503 || error?.status === 429) {
          return failureCount < 3;
        }
        // retry on network errors
        if (error?.name === 'TypeError') {
          return failureCount < 3;
        }
        return false;
      },
      retryDelay: (attemptIndex) => Math.min(1000 * Math.pow(2, attemptIndex), 30000),
    },
  },
});

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <CollectorApp />
    </QueryClientProvider>
  );
}

export default App;
