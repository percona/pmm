import { FC, PropsWithChildren, useState, useCallback, useMemo } from 'react';
import { TourProvider as ReactTourProvider, useTour } from '@reactour/tour';
import { TourContext } from './tour.context';
import { TourName } from './tour.context.types';
import { TOUR_STEPS_MAP } from './tour.constants';
import { TourNavigation } from 'components/tour-navigation';
import { useTheme } from '@mui/material';
import { DRAWER_WIDTH } from 'components/sidebar/drawer/Drawer.constants';

const TourContextProvider: FC<PropsWithChildren> = ({ children }) => {
  const reactTour = useTour();
  const [tourSteps, setTourSteps] = useState(TOUR_STEPS_MAP.product);

  const startTour = useCallback(
    (tourName: TourName) => {
      const steps = TOUR_STEPS_MAP[tourName];
      setTourSteps(steps);
      reactTour.setIsOpen(true);
      reactTour.setCurrentStep(0);
    },
    [reactTour]
  );

  const endTour = useCallback(() => {
    reactTour.setIsOpen(false);
  }, [reactTour]);

  const nextStep = useCallback(() => {
    if (reactTour.currentStep < tourSteps.length - 1) {
      reactTour.setCurrentStep(reactTour.currentStep + 1);
    }
  }, [reactTour, tourSteps.length]);

  const previousStep = useCallback(() => {
    if (reactTour.currentStep > 0) {
      reactTour.setCurrentStep(reactTour.currentStep - 1);
    }
  }, [reactTour]);

  return (
    <TourContext.Provider
      value={{
        startTour,
        endTour,
        nextStep,
        previousStep,
        currentStep: reactTour.currentStep,
        isFirstStep: reactTour.currentStep === 0,
        isLastStep: reactTour.currentStep === tourSteps.length - 1,
      }}
    >
      {children}
    </TourContext.Provider>
  );
};

export const TourProvider: FC<PropsWithChildren> = ({ children }) => {
  const theme = useTheme();

  return (
    <ReactTourProvider
      steps={TOUR_STEPS_MAP.product}
      position="right"
      components={{
        Badge: () => null,
        Close: () => null,
        Navigation: TourNavigation,
      }}
      styles={{
        maskArea: (props) => ({
          ...props,
          width: DRAWER_WIDTH,
          rx: theme.shape.borderRadius,
        }),
        popover: (props) => ({
          ...props,
          padding: theme.spacing(2),
          borderRadius: theme.shape.borderRadius,
          boxShadow: theme.shadows[24],
          width: '480px',
          maxWidth: 'auto',
        }),
      }}
    >
      <TourContextProvider>{children}</TourContextProvider>
    </ReactTourProvider>
  );
};
