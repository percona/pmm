import { fireEvent, render, screen } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { AdreConversationsSidebar } from './AdreConversationsSidebar';

vi.mock('hooks/useAdreChat', () => ({
  formatTimestamp: () => 'Oct 17 10:30',
}));

const props = () => ({
  conversationId: 1,
  conversations: [
    { id: 1, title: 'migration check', lastMessageAt: '2026-10-17T22:30:00Z' },
    { id: 2, title: '', lastMessageAt: '2026-10-16T14:21:00Z' },
  ] as Parameters<typeof AdreConversationsSidebar>[0]['conversations'],
  loading: false,
  searchHits: [],
  searchLoading: false,
  onNewChat: vi.fn(),
  onDeleteConversation: vi.fn(),
  onSelectConversation: vi.fn(),
  onSearch: vi.fn(),
});

describe('AdreConversationsSidebar', () => {
  it('lists chats with their time and starts a new one from the top', () => {
    const p = props();
    render(
      <TestWrapper>
        <AdreConversationsSidebar {...p} />
      </TestWrapper>
    );
    expect(screen.getByText('migration check')).toBeInTheDocument();
    expect(screen.getByText('Chat 2')).toBeInTheDocument();
    expect(screen.getAllByText('Oct 17 10:30')).toHaveLength(2);
    fireEvent.click(screen.getByRole('button', { name: 'New chat' }));
    expect(p.onNewChat).toHaveBeenCalled();
  });

  it('opens and deletes chats', () => {
    const p = props();
    render(
      <TestWrapper>
        <AdreConversationsSidebar {...p} />
      </TestWrapper>
    );
    fireEvent.click(screen.getByText('Chat 2'));
    expect(p.onSelectConversation).toHaveBeenCalledWith(2);
    fireEvent.click(
      screen.getByRole('button', { name: 'Delete conversation Chat 2' })
    );
    expect(p.onDeleteConversation).toHaveBeenCalledWith(2);
  });
});
