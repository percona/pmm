import React, { type ComponentType } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

// One client for all PMM panels so panels on a dashboard share cached data.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: false,
    },
  },
});

export const withQueryClient = <P extends object>(
  Component: ComponentType<P>
) => {
  const WithQueryClient = (props: P) => (
    <QueryClientProvider client={queryClient}>
      <Component {...props} />
    </QueryClientProvider>
  );

  return WithQueryClient;
};
