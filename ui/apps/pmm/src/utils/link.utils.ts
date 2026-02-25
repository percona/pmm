export const constructUrl = (location: {
  pathname: string;
  search: string;
  hash: string;
}) => location.pathname + location.search + location.hash;
