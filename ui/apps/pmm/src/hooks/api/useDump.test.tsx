import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import { startDump } from 'api/dump';
import { PropsWithChildren } from 'react';
import { useStartDump } from './useDump';

vi.mock('api/dump');

describe('useDump', () => {
  it('invalidates the dump list after starting a dump', async () => {
    const queryClient = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    });
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const payload = {
      serviceNames: ['mysql'],
      startTime: '2026-07-10T08:00:00Z',
      endTime: '2026-07-10T10:00:00Z',
      exportQan: false,
      ignoreLoad: true,
      enableEncryption: false,
      encryptionPassword: '',
    };
    const { result } = renderHook(() => useStartDump(), { wrapper });

    await act(() => result.current.mutateAsync(payload));

    expect(vi.mocked(startDump)).toHaveBeenCalledWith(
      payload,
      expect.anything()
    );
    await waitFor(() =>
      expect(invalidate).toHaveBeenCalledWith({
        queryKey: ['dumps:list'],
      })
    );
  });
});
