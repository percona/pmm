import { Location } from 'react-router-dom';

export const constructUrl = (location: Location) =>
  location.pathname + location.search + location.hash;
