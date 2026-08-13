import { createContext } from 'react';
import type { TourContextProps } from './tour.context.types';

export const TourContext = createContext<TourContextProps>({
  startTour: () => {},
  endTour: () => {},
});
