import { useContext } from 'react';
import { TourContext } from './tour.context';

export const useTour = () => useContext(TourContext);
