import { fireEvent, render, screen } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { AdreChatPanel } from './AdreChatPanel';
import type { AlertItem } from './AdreAlertsPanel';

const { handleSend, chatOptions, chatState } = vi.hoisted(() => ({
  handleSend: vi.fn(),
  chatOptions: vi.fn(),
  chatState: { allMessages: [] as unknown[] },
}));

vi.mock('hooks/api/useAdre', () => ({
  useAdreModels: () => ({ data: [], status: 'success' }),
}));

vi.mock('hooks/api/useInvestigations', () => ({
  useCreateInvestigation: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock('hooks/useAdreChat', () => ({
  formatTimestamp: () => '',
  useAdreChat: (options: unknown) => {
    chatOptions(options);
    return {
      response: '',
      reasoning: '',
      loading: false,
      progressSteps: [],
      allMessages: chatState.allMessages,
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
    };
  },
}));

const answered = [
  { role: 'user', content: 'Why does shop_app fail to log in?' },
  {
    role: 'assistant',
    content: 'Failed logins ran at about 100 a minute.',
    progressSteps: [{ id: '1', toolName: 'pmm_logs_patterns', status: 'done' }],
    usage: { model: 'opus5', totalTokens: 467900, totalCost: 0.511 },
  },
];

const alerts: AlertItem[] = [
  { fingerprint: 'f1', labels: { alertname: 'PMM agent down' } },
];

const renderPanel = (path = '/assistant', panelAlerts: AlertItem[] = []) =>
  render(
    wrapWithQueryProvider(
      <TestWrapper routerProps={{ initialEntries: [path] }}>
        <AdreChatPanel alerts={panelAlerts} />
      </TestWrapper>
    )
  );

// MUI's useMediaQuery reads window.matchMedia, which jsdom lacks.
const setWide = (wide: boolean) =>
  vi.stubGlobal(
    'matchMedia',
    vi.fn((query: string) => ({
      matches: wide,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))
  );

describe('AdreChatPanel', () => {
  beforeEach(() => {
    chatState.allMessages = [];
    localStorage.clear();
  });
  afterEach(() => vi.unstubAllGlobals());

  it('prefills the question from ?ask= without sending it', () => {
    renderPanel('/assistant?ask=hello');
    expect(screen.getByDisplayValue('hello')).toBeInTheDocument();
    expect(handleSend).not.toHaveBeenCalled();
  });

  it('starts a new chat for a question from another page', () => {
    renderPanel('/assistant?ask=hello');
    expect(chatOptions).toHaveBeenLastCalledWith({ startFresh: true });
  });

  it('reopens the last chat otherwise', () => {
    renderPanel();
    expect(chatOptions).toHaveBeenLastCalledWith({ startFresh: false });
  });

  it('explains the two modes next to their choice', () => {
    renderPanel();
    expect(
      screen.getByText('Quick answers, lighter analysis')
    ).toBeInTheDocument();
    expect(
      screen.getByText('Deeper analysis with tools and todo steps')
    ).toBeInTheDocument();
    const fast = screen.getByRole('radio', { name: /Fast/ });
    fireEvent.click(fast);
    expect(fast).toBeChecked();
  });

  it("puts the latest answer's steps and usage in the side panel on wide screens", () => {
    setWide(true);
    chatState.allMessages = answered;
    renderPanel('/assistant', alerts);
    const rail = screen.getByRole('complementary', { name: 'Answer details' });
    expect(rail).toHaveTextContent('pmm_logs_patterns');
    expect(rail).toHaveTextContent('$0.511');
    expect(rail).toHaveTextContent('PMM agent down');
    expect(screen.getAllByText('pmm_logs_patterns')).toHaveLength(1);
  });

  it('keeps them with the answer and folds alerts away on narrow screens', () => {
    setWide(false);
    chatState.allMessages = answered;
    renderPanel('/assistant', alerts);
    expect(
      screen.queryByRole('complementary', { name: 'Answer details' })
    ).not.toBeInTheDocument();
    expect(
      screen.getByText('opus5 · 467.9k tokens · $0.511')
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '1 firing alert' })
    ).toBeInTheDocument();
  });
});
