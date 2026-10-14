import { render, screen } from '@testing-library/react';
import FeatureCheck from './FeatureCheck';
import { TestWrapper } from 'utils/testWrapper';
import { measurePageSurface, measureSurface } from 'utils/testUtils';

describe('FeatureCheck', () => {
  it('renders its card on the canvas surface', () => {
    const canvas = measurePageSurface('canvas');
    const paper = measurePageSurface('paper');

    // Guards the assertion below: it only means anything while the two
    // surfaces actually differ.
    expect(canvas).not.toBe(paper);

    const disabled = measureSurface(() => {
      const result = render(
        <TestWrapper>
          <FeatureCheck feature="Alerting" pageTitle="Alerting" />
        </TestWrapper>
      );
      expect(screen.getByText(/Alerting is disabled/)).toBeInTheDocument();

      return result;
    });

    expect(disabled).toBe(canvas);
  });
});
