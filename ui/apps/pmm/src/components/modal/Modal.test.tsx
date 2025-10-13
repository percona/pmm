import { render, screen } from '@testing-library/react';
import { Modal } from './Modal';
import { Typography } from '@mui/material';

describe('Modal', () => {
  it('closes modal via button', () => {
    render(
      <Modal open title="Modal">
        <Typography>Content</Typography>
      </Modal>
    );

    expect(screen.getByTestId('modal-close-button')).toBeDefined();
  });

  it('shows content', () => {
    render(
      <Modal open title="Modal">
        <Typography>Content</Typography>
      </Modal>
    );

    expect(screen.getByText('Content')).toBeDefined();
  });

  it('shows title', () => {
    render(
      <Modal open title="Title">
        <Typography>Content</Typography>
      </Modal>
    );

    expect(screen.getByText('Title')).toBeDefined();
  });
});
