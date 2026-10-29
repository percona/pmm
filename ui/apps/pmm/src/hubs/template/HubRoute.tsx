import type { FC } from 'react';
import { useLocation } from 'react-router-dom';
import { NotFoundPage } from 'pages/not-found';
import { HUBS } from '../hubs.registry';
import { HubPage } from './HubPage';

/** The hub whose path the router matched. */
export const HubRoute: FC = () => {
  const { pathname } = useLocation();
  const path = pathname.replace(/\/+$/, '');
  const config = HUBS.find((hub) => hub.path === path);

  return config ? <HubPage config={config} /> : <NotFoundPage />;
};
