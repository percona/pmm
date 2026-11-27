import { render, screen } from '@testing-library/react';
import Card from './card';

it('should render card', () => {
  render(
    <Card
      dataTestId="test"
      title="test title"
      content="test content"
      cardActions={[
        { variant: 'contained', text: 'test-1' },
        { variant: 'outlined', text: 'test-2' },
      ]}
    />
  );
  expect(screen.getByTestId('test-card-content')).toBeInTheDocument();
  expect(screen.getByTestId('test-card-header')).toBeInTheDocument();
  expect(screen.getByTestId('test-card-content-wrapper')).toBeInTheDocument();
  expect(screen.getByTestId('test-card-actions')).toBeInTheDocument();
  expect(screen.getByTestId('test-1-button')).toBeInTheDocument();
  expect(screen.getByTestId('test-2-button')).toBeInTheDocument();
});

it('should render card without title', () => {
  render(
    <Card
      dataTestId="test"
      content="test content"
      cardActions={[
        { variant: 'contained', text: 'test-1' },
        { variant: 'outlined', text: 'test-2' },
      ]}
    />
  );
  expect(screen.getByTestId('test-card-content')).toBeInTheDocument();
  expect(screen.queryByTestId('test-card-header')).not.toBeInTheDocument();
  expect(screen.getByTestId('test-card-content-wrapper')).toBeInTheDocument();
  expect(screen.getByTestId('test-card-actions')).toBeInTheDocument();
  expect(screen.getByTestId('test-1-button')).toBeInTheDocument();
  expect(screen.getByTestId('test-2-button')).toBeInTheDocument();
});

it('should render card without actions', () => {
  render(<Card dataTestId="test" content="test content" title="test title" />);
  expect(screen.getByTestId('test-card-content')).toBeInTheDocument();
  expect(screen.queryByTestId('test-card-header')).toBeInTheDocument();
  expect(screen.getByTestId('test-card-content-wrapper')).toBeInTheDocument();
  expect(screen.queryByTestId('test-card-actions')).not.toBeInTheDocument();
});

it('should render card only with content', () => {
  render(<Card dataTestId="test" content="test content" />);
  expect(screen.getByTestId('test-card-content')).toBeInTheDocument();
  expect(screen.queryByTestId('test-card-header')).not.toBeInTheDocument();
  expect(screen.getByTestId('test-card-content-wrapper')).toBeInTheDocument();
  expect(screen.queryByTestId('test-card-actions')).not.toBeInTheDocument();
});
