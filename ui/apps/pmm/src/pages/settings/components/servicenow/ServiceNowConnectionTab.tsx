import { FC } from 'react';
import { ExtensionsAuthGate } from 'extensions/ExtensionsAuthGate';
import { ServiceNowConnection } from './ServiceNowConnection';

/**
 * The settings tab wrapper for the ServiceNow connection.
 *
 * Everything under it talks to the side-car, and the side-car's settings router refuses a
 * cookie-only mutation before it validates anything (401), so the tab is held
 * behind the same session exchange the PMM Extensions routes use (PMM-15293). Gating here
 * rather than in `Settings` keeps the exchange off the other tabs.
 */
export const ServiceNowConnectionTab: FC = () => (
  <ExtensionsAuthGate>
    <ServiceNowConnection />
  </ExtensionsAuthGate>
);
