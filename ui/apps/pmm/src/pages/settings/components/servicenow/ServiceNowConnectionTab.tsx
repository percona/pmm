import { FC } from 'react';
import { SepAuthGate } from 'sep/SepAuthGate';
import { ServiceNowConnection } from './ServiceNowConnection';

/**
 * The settings tab wrapper for the ServiceNow connection.
 *
 * Everything under it talks to SEP, and SEP's settings router refuses a
 * cookie-only mutation before it validates anything (401), so the tab is held
 * behind the same session exchange the SEP routes use (PMM-15293). Gating here
 * rather than in `Settings` keeps the exchange off the other tabs.
 */
export const ServiceNowConnectionTab: FC = () => (
  <SepAuthGate>
    <ServiceNowConnection />
  </SepAuthGate>
);
