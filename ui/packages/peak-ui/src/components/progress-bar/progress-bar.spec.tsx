import { render, screen } from '@testing-library/react';
import ProgressBar from './progress-bar';

it('should render', () => {
  const label = 'Using 112.52 CPU (16.7%) of 675.33 CPU in total';
  render(
    <ProgressBar
      label={label}
      buffer={10}
      value={5}
      total={10}
      dataTestId="progress-bar1"
    />
  );
  expect(screen.getByText(label)).toBeInTheDocument();
  expect(screen.getByTestId('progress-bar1')).toBeInTheDocument();
});
