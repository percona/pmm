import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { ComponentProps } from 'react';
import { TestWrapper } from 'utils/testWrapper';
import { DeleteModal } from './DeleteModal';

const renderModal = (props: Partial<ComponentProps<typeof DeleteModal>>) => {
  const onConfirm = vi.fn().mockResolvedValue(undefined);

  render(
    <TestWrapper>
      <DeleteModal
        open
        title="Delete it?"
        message="It goes away."
        confirmLabel="Delete"
        testId="x"
        onConfirm={onConfirm}
        onClose={vi.fn()}
        {...props}
      />
    </TestWrapper>
  );

  return { onConfirm };
};

describe('DeleteModal', () => {
  it('renders the force checkbox only with a force description', () => {
    const { unmount } = render(
      <TestWrapper>
        <DeleteModal
          open
          title="Delete it?"
          message="It goes away."
          confirmLabel="Delete"
          testId="x"
          onConfirm={vi.fn()}
          onClose={vi.fn()}
        />
      </TestWrapper>
    );

    expect(screen.queryByTestId('x-force')).not.toBeInTheDocument();
    unmount();

    renderModal({ forceDescription: 'Also removes the agents.' });

    expect(screen.getByTestId('x-force')).toBeInTheDocument();
    expect(screen.getByText('Also removes the agents.')).toBeVisible();
  });

  it('renders children under the message', () => {
    renderModal({ children: <li>item-1</li> });

    expect(screen.getByText('item-1')).toBeVisible();
  });

  it('passes the force choice to onConfirm', async () => {
    const { onConfirm } = renderModal({ forceDescription: 'Force it.' });

    fireEvent.click(screen.getByTestId('x-force'));
    fireEvent.click(screen.getByTestId('x-confirm'));

    await waitFor(() => expect(onConfirm).toHaveBeenCalledWith(true));
  });

  it('builds test ids from testId', () => {
    renderModal({});

    ['x', 'x-title', 'x-message', 'x-cancel', 'x-confirm'].forEach((id) =>
      expect(screen.getByTestId(id)).toBeInTheDocument()
    );
  });
});
