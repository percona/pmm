import { render, screen } from '@testing-library/react';
import AlertDescription from './AlertDescription';

// The default normalizer collapses whitespace, which would hide indentation.
const exact = { normalizer: (value: string) => value };

describe('AlertDescription', () => {
  it('renders unavailable text when the description is missing', () => {
    render(<AlertDescription />);

    expect(screen.getByText('Unavailable')).toBeInTheDocument();
  });

  it('renders each line separately and keeps blank lines', () => {
    const { container } = render(
      <AlertDescription description={'First paragraph.\n\nSecond paragraph.'} />
    );

    const lines = Array.from(container.firstElementChild?.children ?? []);
    expect(lines).toHaveLength(3);
    expect(lines[0]).toHaveTextContent('First paragraph.');
    expect(lines[1].textContent).toBe('\u00a0');
    expect(lines[2]).toHaveTextContent('Second paragraph.');
  });

  it('keeps the indentation of a plain line', () => {
    render(<AlertDescription description="  indented note" />);

    expect(screen.getByText('  indented note', exact)).toHaveStyle({
      whiteSpace: 'pre-wrap',
    });
  });

  it.each([
    ['  4. ', 'Confirm PMM Server can reach PMM_CLICKHOUSE_ADDR.'],
    ['2) ', 'Restart the service.'],
    ['  - ', 'Check free disk space.'],
    ['* ', 'Review the logs.'],
    ['• ', 'Upgrade to a patched version.'],
  ])(
    'puts the list marker %j in its own column next to the text',
    (marker, text) => {
      render(<AlertDescription description={`${marker}${text}`} />);

      const markerElement = screen.getByText(marker, exact);
      expect(markerElement).toHaveStyle({ whiteSpace: 'pre', flexShrink: '0' });
      expect(markerElement.nextElementSibling).toHaveTextContent(text);
    }
  );

  it('keeps whitespace inside list item text', () => {
    render(<AlertDescription description={'  1. Run:\tpmm-admin   list'} />);

    expect(screen.getByText('Run:\tpmm-admin   list', exact)).toHaveStyle({
      whiteSpace: 'pre-wrap',
    });
  });

  it('does not treat a version number as a list marker', () => {
    render(<AlertDescription description="5.0 is end of life" />);

    expect(screen.getByText('5.0 is end of life')).toHaveStyle({
      whiteSpace: 'pre-wrap',
    });
  });
});
