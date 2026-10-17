import { render, screen } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { AdreChatPanel } from './AdreChatPanel';

const { handleSend } = vi.hoisted(() => ({ handleSend: vi.fn() }));

vi.mock('hooks/api/useAdre', () => ({
  useAdreModels: () => ({ data: [], status: 'success' }),
}));

vi.mock('hooks/useAdreChat', () => ({
  formatTimestamp: () => '',
  useAdreChat: () => ({
    response: '',
    reasoning: '',
    loading: false,
    progressSteps: [],
    allMessages: [],
    settings: undefined,
    chatError: null,
    handleSend,
    conversationId: null,
    conversations: [],
    conversationsLoading: false,
    newChat: vi.fn(),
    deleteConversation: vi.fn(),
    selectConversation: vi.fn(),
    searchHits: [],
    searchLoading: false,
    runSearch: vi.fn(),
    scrollToMessageId: null,
    clearScrollToMessage: vi.fn(),
  }),
}));

describe('AdreChatPanel', () => {
  it('prefills the question from ?ask= without sending it', () => {
    render(
      wrapWithQueryProvider(
        <TestWrapper routerProps={{ initialEntries: ['/assistant?ask=hello'] }}>
          <AdreChatPanel />
        </TestWrapper>
      )
    );
    expect(screen.getByDisplayValue('hello')).toBeInTheDocument();
    expect(handleSend).not.toHaveBeenCalled();
  });
});
