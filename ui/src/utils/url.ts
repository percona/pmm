import { Location } from 'react-router-dom';

export const constructUrlFromLocation = (location: Location): string => {
  return location.pathname + location.search + location.hash;
};
