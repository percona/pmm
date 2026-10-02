import { render, screen } from '@testing-library/react';
import AlertDescription from './AlertDescription';

// The default normalizer collapses whitespace, which would hide the bug.
const exact = { normalizer: (value: string) => value };

describe('AlertDescription', () => {
  it('renders unavailable text when the description is missing', () => {
    render(<AlertDescription />);

    expect(screen.getByText('Unavailable')).toBeInTheDocument();
  });

  it('keeps line breaks and indentation of the description', () => {
    const description =
      'ClickHouse is not responding.\n\nRemediation steps:\n  1. Restart it.\n  2. Check free disk space.';

    render(<AlertDescription description={description} />);

    expect(screen.getByText(description, exact)).toHaveStyle({
      whiteSpace: 'pre-wrap',
    });
  });
});
