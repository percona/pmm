import { QueryClient } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import { NodeType } from 'types/nodes.types';
import { DiscoverAzureDatabaseType } from 'types/services.types';
import { wrapWithQueryProvider } from 'utils/testUtils';
import {
  useAddAzureService,
  useAddService,
  useServerRDSInstances,
} from './useServices';

const { addService, addAzureService, discoverRDS } = vi.hoisted(() => ({
  addService: vi.fn(),
  addAzureService: vi.fn(),
  discoverRDS: vi.fn(),
}));

vi.mock('api/services', () => ({ addService, addAzureService, discoverRDS }));

const INVENTORY_KEYS = [
  ['services:getTypes'],
  ['services:list'],
  ['services:managed'],
  ['nodes:list'],
];

const renderWithClient = <T,>(hook: () => T) => {
  const queryClient = new QueryClient();
  const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
  const { result } = renderHook(hook, {
    wrapper: ({ children }) => wrapWithQueryProvider(children, queryClient),
  });
  return { result, invalidate };
};

describe('useAddService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('refreshes the inventory and calls onSuccess', async () => {
    addService.mockResolvedValue(undefined);
    const onSuccess = vi.fn();
    const { result, invalidate } = renderWithClient(() =>
      useAddService({ onSuccess })
    );

    await act(() =>
      result.current.mutateAsync({
        external: {
          serviceName: 'redis',
          address: 'redis',
          listenPort: 9121,
          addNode: { nodeName: 'redis', nodeType: NodeType.remote },
        },
      })
    );

    expect(invalidate.mock.calls.map(([filters]) => filters?.queryKey)).toEqual(
      INVENTORY_KEYS
    );
    expect(onSuccess).toHaveBeenCalledTimes(1);
  });

  it('leaves the inventory alone when adding fails', async () => {
    addService.mockRejectedValue(new Error('connection refused'));
    const { result, invalidate } = renderWithClient(() => useAddService());

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          haproxy: {
            serviceName: 'haproxy',
            address: 'haproxy',
            listenPort: 8404,
          },
        })
      ).rejects.toThrow('connection refused');
    });

    expect(invalidate).not.toHaveBeenCalled();
  });
});

describe('useAddAzureService', () => {
  it('refreshes the inventory', async () => {
    addAzureService.mockResolvedValue(undefined);
    const { result, invalidate } = renderWithClient(() => useAddAzureService());

    await act(() =>
      result.current.mutateAsync({
        region: 'eastus',
        instanceId: 'azure-1',
        address: 'azure-1',
        port: 5432,
        username: 'admin',
        azureClientId: 'client',
        azureClientSecret: 'secret',
        azureTenantId: 'tenant',
        azureSubscriptionId: 'subscription',
        azureResourceGroup: 'group',
        type: DiscoverAzureDatabaseType.postgresql,
      })
    );

    expect(invalidate.mock.calls.map(([filters]) => filters?.queryKey)).toEqual(
      INVENTORY_KEYS
    );
  });
});

describe('useServerRDSInstances', () => {
  it('cancels the discovery when nothing uses it any more', async () => {
    discoverRDS.mockReturnValue(new Promise(() => {}));
    const { unmount } = renderHook(() => useServerRDSInstances(), {
      wrapper: ({ children }) => wrapWithQueryProvider(children),
    });
    await waitFor(() => expect(discoverRDS).toHaveBeenCalledTimes(1));
    const [payload, signal] = discoverRDS.mock.calls[0] as [
      object,
      AbortSignal,
    ];
    expect(payload).toEqual({});
    expect(signal.aborted).toBe(false);

    unmount();

    expect(signal.aborted).toBe(true);
  });
});
