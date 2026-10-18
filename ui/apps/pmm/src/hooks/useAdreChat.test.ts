import { renderHook, waitFor } from '@testing-library/react';
import { useAdreChat } from './useAdreChat';

const { listAdreConversations, getAdreMessages } = vi.hoisted(() => ({
  listAdreConversations: vi.fn(),
  getAdreMessages: vi.fn(),
}));

vi.mock('api/adre', async (importOriginal) => ({
  ...(await importOriginal<typeof import('api/adre')>()),
  listAdreConversations,
  getAdreMessages,
}));

vi.mock('hooks/api/useAdre', () => ({
  useAdreSettings: () => ({ data: { enabled: true } }),
}));

vi.mock('notistack', () => ({
  useSnackbar: () => ({ enqueueSnackbar: vi.fn() }),
}));

describe('useAdreChat', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
    listAdreConversations.mockResolvedValue({
      conversations: [{ id: 5 }, { id: 4 }],
    });
    getAdreMessages.mockResolvedValue({ messages: [] });
  });

  it('reopens the latest conversation', async () => {
    const { result } = renderHook(() => useAdreChat());
    await waitFor(() => expect(result.current.conversationId).toBe(5));
  });

  it('opens no conversation when starting fresh', async () => {
    const { result } = renderHook(() => useAdreChat({ startFresh: true }));
    await waitFor(() => expect(result.current.conversations).toHaveLength(2));
    expect(result.current.conversationId).toBeNull();
    expect(getAdreMessages).not.toHaveBeenCalled();
  });
});
