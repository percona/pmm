export type TourName = 'product';

export interface TourContextProps {
  startTour: (tourName: TourName) => void;
  endTour: () => void;
  nextStep: () => void;
  previousStep: () => void;
  currentStep: number;
  isFirstStep: boolean;
  isLastStep: boolean;
}
