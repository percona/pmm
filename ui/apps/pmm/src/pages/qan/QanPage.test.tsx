import { render } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi } from 'vitest';
import QanPage from './QanPage';

vi.mock('contexts/settings', () => ({
  useSettings: () => ({ settings: { nativeQanEnabled: true } }),
}));

vi.mock('hooks/api/useAdre', () => ({
  useAdreSettings: () => ({ data: { enabled: false } }),
}));

vi.mock('hooks/useAdreChat', () => ({
  useAdreChat: () => ({
    response: '',
    reasoning: '',
    loading: false,
    progressSteps: [],
    allMessages: [],
    chatError: null,
    handleSend: vi.fn(),
    resetEphemeralChat: vi.fn(),
  }),
  formatTimestamp: () => '',
}));

vi.mock('hooks/api/useQan', () => ({
  useQanReport: () => ({ data: { rows: [], totalRows: 0 }, isLoading: false, isError: false }),
  useQanFilters: () => ({ data: { labels: {} }, isLoading: false }),
  useQanMetricNames: () => ({ data: { data: [] } }),
}));

vi.mock('components/page', () => ({
  Page: ({ children }: { children: React.ReactNode }) => <div data-testid="qan-page">{children}</div>,
}));

const wrapper = ({ children }: { children: React.ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/pmm-ui/qan']}>{children}</MemoryRouter>
    </QueryClientProvider>
  );
};

describe('QanPage', () => {
  it('renders native QAN layout shell', () => {
    const { getByTestId } = render(<QanPage />, { wrapper });
    expect(getByTestId('qan-controls')).toBeInTheDocument();
    expect(getByTestId('qan-ai-aside')).toBeInTheDocument();
  });
});
