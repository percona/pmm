import { fireEvent, render, screen } from '@testing-library/react';
import PreviewChip from './PreviewChip';
import type { PreviewLevel } from './PreviewChip.types';

describe('PreviewChip', () => {
  it('renders the chip', () => {
    render(<PreviewChip level="tech" />);

    expect(screen.getByText('Preview')).toBeInTheDocument();
  });

  it.each<[PreviewLevel, string]>([
    ['tech', 'Tech preview'],
    ['developer', 'Developer preview'],
  ])('names the %s level on hover', async (level, tooltip) => {
    render(<PreviewChip level={level} />);

    fireEvent.mouseOver(screen.getByTestId('preview-chip'));

    expect(await screen.findByRole('tooltip')).toHaveTextContent(tooltip);
  });

  it('keeps "Preview" as the accessible name of the chip', () => {
    render(<PreviewChip level="tech" />);

    expect(screen.getByTestId('preview-chip')).not.toHaveAttribute(
      'aria-label'
    );
  });
});
