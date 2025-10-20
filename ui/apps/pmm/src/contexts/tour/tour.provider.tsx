import {
  FC,
  PropsWithChildren,
  useState,
  useCallback,
  useMemo,
  useEffect,
} from 'react';
import { TourProvider as ReactTourProvider, StepType } from '@reactour/tour';
import { TourContext } from './tour.context';
import { StepsMap, TourName } from './tour.context.types';
import { TourNavigation } from 'components/tour-navigation';
import { useTheme } from '@mui/material';
import { DRAWER_WIDTH } from 'components/sidebar/drawer/Drawer.constants';
import { waitForVisible } from 'utils/dom.utils';
import { useUpdateUserInfo } from 'hooks/api/useUser';
import { useUser } from 'contexts/user';
import { getProductTourSteps } from './steps/product.steps';
import { getAlertingTourSteps } from './steps/alerting.steps';
import { TourCloseButton } from 'components/tour-close-button';
import { useNavigation } from 'contexts/navigation';
import { useLocation } from 'react-router-dom';

export const TourProvider: FC<PropsWithChildren> = ({ children }) => {
  const theme = useTheme();
  const { user } = useUser();
  const [tourName, setTourName] = useState<TourName>('product');
  const { mutateAsync } = useUpdateUserInfo();
  const stepsMap = useMemo<StepsMap>(
    () => ({
      product: getProductTourSteps(user),
      alerting: getAlertingTourSteps(user),
    }),
    [user?.isPMMAdmin]
  );
  const [isOpen, setIsOpen] = useState(false);
  const [currentStep, setCurrentStep] = useState(0);
  const [steps, setSteps] = useState<StepType[]>([]);
  const { navOpen, setNavOpen } = useNavigation();
  const location = useLocation();

  const startTour = useCallback(
    async (tourName: TourName) => {
      const steps = stepsMap[tourName];

      if (!navOpen) {
        setNavOpen(true);
      }

      if (steps[0].selector && typeof steps[0].selector === 'string') {
        await waitForVisible(steps[0].selector);
      }

      setTourName(tourName);
      setCurrentStep(0);
      setSteps(steps);
      setIsOpen(true);
    },
    [stepsMap, navOpen, setNavOpen]
  );

  const endTour = useCallback(async () => {
    if (tourName === 'alerting') {
      await mutateAsync({ alertingTourCompleted: true });
    } else if (tourName === 'product') {
      await mutateAsync({ productTourCompleted: true });
    }
    setIsOpen(false);
  }, [tourName]);

  useEffect(() => {
    if (isOpen || !user?.info) {
      return;
    }

    if (
      !user.info.alertingTourCompleted &&
      location.pathname.includes('/alerting')
    ) {
      startTour('alerting');
    }
  }, [isOpen, location.pathname, user?.info]);

  return (
    <TourContext.Provider
      value={{
        startTour,
        endTour,
      }}
    >
      {/* Need to conditionaly render the provider since the defaultOpen property doesn't react on isOpen change */}
      {isOpen ? (
        <ReactTourProvider
          defaultOpen
          steps={steps}
          currentStep={currentStep}
          setCurrentStep={setCurrentStep}
          onClickClose={endTour}
          onClickMask={endTour}
          position="right"
          components={{
            Badge: () => null,
            Close: () => <TourCloseButton endTour={endTour} />,
            Navigation: () => (
              <TourNavigation
                currentStep={currentStep}
                setCurrentStep={setCurrentStep}
                stepCount={steps.length}
                endTour={endTour}
              />
            ),
          }}
          styles={{
            maskArea: (props) => ({
              ...props,
              width: tourName === 'alerting' ? DRAWER_WIDTH - 22 : DRAWER_WIDTH,
              rx: theme.shape.borderRadius,
            }),
            popover: (props) => ({
              ...props,
              padding: theme.spacing(2),
              backgroundColor: theme.palette.background.paper,
              borderRadius: theme.shape.borderRadius,
              boxShadow: theme.shadows[24],
              width: '480px',
              maxWidth: 'auto',
            }),
          }}
        >
          {children}
        </ReactTourProvider>
      ) : (
        children
      )}
    </TourContext.Provider>
  );
};
