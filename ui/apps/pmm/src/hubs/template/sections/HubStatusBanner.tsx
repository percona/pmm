import ArrowDownward from '@mui/icons-material/ArrowDownward';
import ArrowForward from '@mui/icons-material/ArrowForward';
import { StateIndicator, StatusBanner, StatusCard } from '@percona/peak-ui';
import { ThemeScope } from 'components/theme-scope';
import type { FC } from 'react';
import { useLocation } from 'react-router-dom';
import { Messages } from '../hub.messages';
import { type HubStatusContext, statusFacets } from '../hub.status';
import type { HubDoor, HubStatus } from '../hub.types';
import { DoorButton } from './DoorLink';

export interface HubStatusBannerProps extends HubStatusContext {
  status: HubStatus;
  /** Before an action takes the reader down to Needs attention. */
  onReview?: () => void;
}

/** A facet's action: down to a section of this page, keeping the hub's params, or on to another page. */
const FacetAction: FC<{ door: HubDoor; onInPage?: () => void }> = ({
  door,
  onInPage,
}) => {
  const { search } = useLocation();
  const inPage = door.to.startsWith('#');

  return (
    <DoorButton
      door={inPage ? { ...door, to: `${search}${door.to}` } : door}
      endIcon={inPage ? <ArrowDownward /> : <ArrowForward />}
      {...(inPage && onInPage ? { onClick: onInPage } : {})}
    />
  );
};

/** What to look at first: cluster health, alerts and backups, on cards when there is something to do. */
export const HubStatusBanner: FC<HubStatusBannerProps> = ({
  status,
  onReview,
  ...context
}) => (
  // Neutral, Peak's Base theme, as the Figma hub draws the cards' actions and the header's Add.
  <ThemeScope theme="base">
    <StatusBanner
      aria-label={Messages.banner.label(context.name)}
      data-testid="hub-status-banner"
    >
      {statusFacets(status, context).map(({ id, state, label, action }) =>
        action ? (
          <StatusCard
            key={id}
            state={state}
            action={<FacetAction door={action} onInPage={onReview} />}
          >
            {label}
          </StatusCard>
        ) : (
          <StateIndicator key={id} state={state}>
            {label}
          </StateIndicator>
        )
      )}
    </StatusBanner>
  </ThemeScope>
);
