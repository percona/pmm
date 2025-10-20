import { createContext } from 'react';
import { TourContextProps } from './tour.context.types';

export const TourContext = createContext<TourContextProps>({
  startTour: () => {},
  endTour: () => {},
});
