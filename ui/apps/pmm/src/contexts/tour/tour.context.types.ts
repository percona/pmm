import { StepType } from '@reactour/tour';

export type TourName = 'product' | 'alerting';

export type StepsMap = Record<TourName, StepType[]>;

export interface TourContextProps {
  startTour: (tourName: TourName) => void;
  endTour: () => void;
}
