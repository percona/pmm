import { renderHook, waitFor } from '@testing-library/react';
import { mapServerRowsToChat, useAdreChat } from './useAdreChat';

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

describe('mapServerRowsToChat', () => {
  const row = (id: number, role: string, extra: object = {}) => ({
    id,
    conversationId: 5,
    role,
    content: role === 'tool' ? '' : `${role} ${id}`,
    createdAt: '2026-10-17T21:00:00Z',
    ...extra,
  });
  const tool = (id: number, callId: string) =>
    row(id, 'tool', {
      toolName: 'pmm_clickhouse_query',
      toolResultJson: {
        toolCallId: callId,
        result: { params: { query: 'SELECT 1' } },
      },
    });

  it('gives each answer the tool steps of its turn, saved before or after it', () => {
    const chat = mapServerRowsToChat([
      // A window that starts mid-turn: the tool belongs to the answer that follows.
      tool(1, 'early'),
      row(2, 'assistant'),
      row(3, 'user'),
      row(4, 'assistant'),
      tool(5, 'a'),
      tool(6, 'b'),
      row(7, 'user'),
      row(8, 'assistant'),
    ]);

    expect(chat.map((m) => m.role)).toEqual([
      'assistant',
      'user',
      'assistant',
      'user',
      'assistant',
    ]);
    expect(chat[0].progressSteps?.map((s) => s.id)).toEqual(['early']);
    expect(chat[2].progressSteps?.map((s) => s.id)).toEqual(['a', 'b']);
    expect(chat[2].progressSteps?.[0].params).toEqual({ query: 'SELECT 1' });
    expect(chat[4].progressSteps).toBeUndefined();
  });
});
