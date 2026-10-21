import { StateIndicator, StatusBanner } from '@percona/peak-ui';
import { Fragment, type FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { Messages } from '../hub.messages';
import { type HubStatusContext, statusFacets } from '../hub.status';
import type { HubStatus } from '../hub.types';

export interface HubStatusBannerProps extends HubStatusContext {
  status: HubStatus;
}

/** What to look at first: cluster health, alerts and backups, each with a way to set up what is missing. */
export const HubStatusBanner: FC<HubStatusBannerProps> = ({
  status,
  ...context
}) => (
  <StatusBanner
    aria-label={Messages.banner.label(context.name)}
    data-testid="hub-status-banner"
  >
    {statusFacets(status, context).map((facet) => (
      <Fragment key={facet.id}>
        <StateIndicator state={facet.state}>{facet.label}</StateIndicator>
        {facet.action && (
          <StateIndicator
            state="actionable"
            component={RouterLink}
            to={facet.action.to}
          >
            {facet.action.label}
          </StateIndicator>
        )}
      </Fragment>
    ))}
  </StatusBanner>
);
