import { fireEvent, render, screen } from '@testing-library/react';
import { ExportButton } from './ExportButton';

describe('ExportButton', () => {
  it('exports on click', () => {
    const onClick = vi.fn();
    render(<ExportButton onClick={onClick} />);

    fireEvent.click(screen.getByRole('button', { name: 'Export' }));

    expect(onClick).toHaveBeenCalled();
  });
});
