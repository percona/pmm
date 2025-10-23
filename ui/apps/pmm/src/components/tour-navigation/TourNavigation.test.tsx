import { render, screen, fireEvent } from '@testing-library/react';
import TourNavigation from './TourNavigation';

describe('TourNavigation', () => {
  const mockSetCurrentStep = vi.fn();
  const mockEndTour = vi.fn();

  beforeEach(() => {
    mockSetCurrentStep.mockClear();
    mockEndTour.mockClear();
  });

  describe('Navigation buttons', () => {
    it('does not show previous button on first step', () => {
      render(
        <TourNavigation
          currentStep={0}
          setCurrentStep={mockSetCurrentStep}
          stepCount={5}
          endTour={mockEndTour}
        />
      );

      expect(screen.queryByTestId('tour-previous-step-button')).toBeNull();
    });

    it('shows previous button on second step', () => {
      render(
        <TourNavigation
          currentStep={1}
          setCurrentStep={mockSetCurrentStep}
          stepCount={5}
          endTour={mockEndTour}
        />
      );

      expect(screen.getByTestId('tour-previous-step-button')).toBeDefined();
    });

    it('shows next button on first step', () => {
      render(
        <TourNavigation
          currentStep={0}
          setCurrentStep={mockSetCurrentStep}
          stepCount={5}
          endTour={mockEndTour}
        />
      );

      expect(screen.getByTestId('tour-next-step-button')).toBeDefined();
    });

    it('shows end tour button on last step', () => {
      render(
        <TourNavigation
          currentStep={4}
          setCurrentStep={mockSetCurrentStep}
          stepCount={5}
          endTour={mockEndTour}
        />
      );

      expect(screen.getByTestId('tour-end-tour-button')).toBeDefined();
      expect(screen.queryByTestId('tour-next-step-button')).toBeNull();
    });
  });

  describe('Step counter', () => {
    it('displays correct step counter on first step', () => {
      render(
        <TourNavigation
          currentStep={0}
          setCurrentStep={mockSetCurrentStep}
          stepCount={5}
          endTour={mockEndTour}
        />
      );

      expect(screen.getByText('Tip 1 of 5')).toBeDefined();
    });

    it('displays correct step counter on middle step', () => {
      render(
        <TourNavigation
          currentStep={2}
          setCurrentStep={mockSetCurrentStep}
          stepCount={5}
          endTour={mockEndTour}
        />
      );

      expect(screen.getByText('Tip 3 of 5')).toBeDefined();
    });

    it('displays correct step counter on last step', () => {
      render(
        <TourNavigation
          currentStep={4}
          setCurrentStep={mockSetCurrentStep}
          stepCount={5}
          endTour={mockEndTour}
        />
      );

      expect(screen.getByText('Tip 5 of 5')).toBeDefined();
    });

    it('displays correct step counter with single step', () => {
      render(
        <TourNavigation
          currentStep={0}
          setCurrentStep={mockSetCurrentStep}
          stepCount={1}
          endTour={mockEndTour}
        />
      );

      expect(screen.getByText('Tip 1 of 1')).toBeDefined();
    });
  });

  describe('User interactions', () => {
    it('calls setCurrentStep with next step when next button is clicked', () => {
      render(
        <TourNavigation
          currentStep={0}
          setCurrentStep={mockSetCurrentStep}
          stepCount={5}
          endTour={mockEndTour}
        />
      );

      fireEvent.click(screen.getByTestId('tour-next-step-button'));

      expect(mockSetCurrentStep).toHaveBeenCalledWith(1);
      expect(mockSetCurrentStep).toHaveBeenCalledTimes(1);
    });

    it('calls setCurrentStep with previous step when previous button is clicked', () => {
      render(
        <TourNavigation
          currentStep={2}
          setCurrentStep={mockSetCurrentStep}
          stepCount={5}
          endTour={mockEndTour}
        />
      );

      fireEvent.click(screen.getByTestId('tour-previous-step-button'));

      expect(mockSetCurrentStep).toHaveBeenCalledWith(1);
      expect(mockSetCurrentStep).toHaveBeenCalledTimes(1);
    });

    it('calls endTour when end tour button is clicked', () => {
      render(
        <TourNavigation
          currentStep={4}
          setCurrentStep={mockSetCurrentStep}
          stepCount={5}
          endTour={mockEndTour}
        />
      );

      fireEvent.click(screen.getByTestId('tour-end-tour-button'));

      expect(mockEndTour).toHaveBeenCalledTimes(1);
      expect(mockSetCurrentStep).not.toHaveBeenCalled();
    });

    it('navigates through multiple steps correctly', () => {
      const { rerender } = render(
        <TourNavigation
          currentStep={0}
          setCurrentStep={mockSetCurrentStep}
          stepCount={3}
          endTour={mockEndTour}
        />
      );

      // Click next from step 1
      fireEvent.click(screen.getByTestId('tour-next-step-button'));
      expect(mockSetCurrentStep).toHaveBeenCalledWith(1);

      // Simulate moving to step 2
      rerender(
        <TourNavigation
          currentStep={1}
          setCurrentStep={mockSetCurrentStep}
          stepCount={3}
          endTour={mockEndTour}
        />
      );

      // Click next from step 2
      fireEvent.click(screen.getByText('Next tip'));
      expect(mockSetCurrentStep).toHaveBeenCalledWith(2);
    });
  });
});
