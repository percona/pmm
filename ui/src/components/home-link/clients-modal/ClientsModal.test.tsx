import { fireEvent, render, screen } from '@testing-library/react';
import { ClientsModal } from './ClientsModal';
import { PMM_HOME_URL } from 'constants';

const onCloseMock = vi.fn();

describe('ClientsModal', () => {
  beforeEach(() => {
    onCloseMock.mockClear();
  });

  it('closes when close icon is clicked', () => {
    render(<ClientsModal isOpen onClose={onCloseMock} />);

    fireEvent.click(screen.getByTestId('modal-close-button'));

    expect(onCloseMock).toHaveBeenCalled();
  });

  it('closes when close window button is clicked', () => {
    render(<ClientsModal isOpen onClose={onCloseMock} />);

    fireEvent.click(screen.getByTestId('modal-close-window-button'));

    expect(onCloseMock).toHaveBeenCalled();
  });

  it('navigates home when "Go to PMM Home" is clicked', () => {
    render(<ClientsModal isOpen onClose={onCloseMock} />);

    expect(screen.getByTestId('modal-pmm-home-link')).toHaveAttribute(
      'href',
      PMM_HOME_URL
    );
  });
});
