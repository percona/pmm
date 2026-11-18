import { renderHook, waitFor } from '@testing-library/react';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { useHasKubernetesData } from './useKubernetes';

const { hasKubernetesData } = vi.hoisted(() => ({
  hasKubernetesData: vi.fn(),
}));

vi.mock('api/kubernetes', () => ({ hasKubernetesData }));

describe('useHasKubernetesData', () => {
  beforeEach(() => {
    hasKubernetesData.mockReset();
  });

  it('reports the API answer', async () => {
    hasKubernetesData.mockResolvedValue(true);

    const { result } = renderHook(() => useHasKubernetesData(), {
      wrapper: ({ children }) => wrapWithQueryProvider(children),
    });

    await waitFor(() => expect(result.current.data).toBe(true));
  });

  it('does not query without a user', () => {
    renderHook(() => useHasKubernetesData({ enabled: false }), {
      wrapper: ({ children }) => wrapWithQueryProvider(children),
    });

    expect(hasKubernetesData).not.toHaveBeenCalled();
  });
});
