import { render, screen } from '@testing-library/react';
import TourStep from './TourStep';
import Typography from '@mui/material/Typography';

describe('TourStep', () => {
  it('renders the title', () => {
    render(<TourStep title="Test Title">Content</TourStep>);

    expect(screen.getByTestId('tour-step-title')).toBeDefined();
    expect(screen.getByText('Test Title')).toBeDefined();
  });

  it('renders children content', () => {
    render(
      <TourStep title="Test Title">
        <Typography>Test content</Typography>
      </TourStep>
    );

    expect(screen.getByText('Test content')).toBeDefined();
  });

  it('renders multiple children', () => {
    render(
      <TourStep title="Test Title">
        <Typography>First paragraph</Typography>
        <Typography>Second paragraph</Typography>
        <Typography>Third paragraph</Typography>
      </TourStep>
    );

    expect(screen.getByText('First paragraph')).toBeDefined();
    expect(screen.getByText('Second paragraph')).toBeDefined();
    expect(screen.getByText('Third paragraph')).toBeDefined();
  });

  it('renders the tour step body container', () => {
    render(
      <TourStep title="Test Title">
        <Typography>Content</Typography>
      </TourStep>
    );

    expect(screen.getByTestId('tour-step-body')).toBeDefined();
  });

  it('renders with empty children', () => {
    render(<TourStep title="Test Title" />);

    expect(screen.getByTestId('tour-step-title')).toBeDefined();
    expect(screen.getByTestId('tour-step-body')).toBeDefined();
  });

  it('renders title with h5 variant', () => {
    render(<TourStep title="Test Title">Content</TourStep>);

    const title = screen.getByTestId('tour-step-title');
    expect(title.tagName).toBe('H5');
  });

  it('renders with complex nested children', () => {
    render(
      <TourStep title="Complex Step">
        <div>
          <Typography>Nested content</Typography>
          <ul>
            <li>Item 1</li>
            <li>Item 2</li>
          </ul>
        </div>
      </TourStep>
    );

    expect(screen.getByText('Nested content')).toBeDefined();
    expect(screen.getByText('Item 1')).toBeDefined();
    expect(screen.getByText('Item 2')).toBeDefined();
  });
});
