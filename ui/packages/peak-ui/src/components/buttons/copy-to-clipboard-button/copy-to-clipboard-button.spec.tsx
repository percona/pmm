import { afterEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import CopyToClipboardButton from '.';
import { Messages } from './clipboard.messages';

const mockClipboard = () => {
  const writeText = vi.fn().mockResolvedValue(undefined);
  Object.defineProperty(navigator, 'clipboard', {
    value: { writeText },
    configurable: true,
  });
  return writeText;
};

afterEach(() => {
  // jsdom has no clipboard by default; remove any mock so tests stay isolated
  delete (navigator as { clipboard?: unknown }).clipboard;
});

describe('CopyToClipboardButton', () => {
  it('renders a disabled icon button explaining restricted access when the clipboard is unavailable', () => {
    const { container } = render(<CopyToClipboardButton textToCopy="text" />);
    const button = container.querySelector('.MuiIconButton-root');

    expect(button).not.toBeNull();
    expect(button).toHaveClass('Mui-disabled');
    expect(button!.parentElement).toHaveAttribute('aria-label', Messages.restrictedAccess);
  });

  it('keeps the disabled button hoverable when consumers pass their own sx', () => {
    const { container } = render(
      <CopyToClipboardButton textToCopy="text" buttonProps={{ sx: { marginTop: '7px' } }} />
    );
    const button = container.querySelector('.MuiIconButton-root') as HTMLElement;

    // both the built-in rule and the consumer sx must survive the merge
    expect(button).toHaveStyle({ pointerEvents: 'auto' });
    expect(button).toHaveStyle({ marginTop: '7px' });
  });

  it('copies the text and confirms with a tooltip', () => {
    const writeText = mockClipboard();
    const { container } = render(<CopyToClipboardButton textToCopy="pnpm build" />);
    const button = container.querySelector('.MuiIconButton-root') as HTMLElement;

    expect(button).not.toHaveClass('Mui-disabled');
    fireEvent.click(button);

    expect(writeText).toHaveBeenCalledWith('pnpm build');
    expect(screen.getByText(Messages.copied)).toBeInTheDocument();
  });

  it('renders a labeled themed button when showCopyButtonText is set', () => {
    mockClipboard();
    render(
      <CopyToClipboardButton
        textToCopy="text"
        showCopyButtonText
        copyCommand="Copy command"
        buttonProps={{ size: 'small' }}
      />
    );
    const button = screen.getByRole('button', { name: /Copy command/ });

    expect(button.tagName).toBe('BUTTON');
    expect(button).toHaveClass('MuiButton-sizeSmall');
  });
});
