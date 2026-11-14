import { StepType } from '@reactour/tour';
import { Messages } from './alerting.messages';
import { TourStep } from 'components/tour-step';
import Typography from '@mui/material/Typography';
import { NavItem } from 'types/navigation.types';
import { NAV_ALERTS } from 'contexts/navigation/navigation.constants';

// Built from the rendered Alerts menu so order, labels and role gating match it.
export const getAlertingTourSteps = (navTree: NavItem[]): StepType[] => {
  const alerts = navTree.find((item) => item.id === NAV_ALERTS.id);

  return (alerts?.children || [])
    .filter((item) => !item.hidden && Messages[item.id])
    .map((item) => ({
      selector: `[data-testid="navitem-${item.id}-list-item"]`,
      content: (
        <TourStep title={item.text || ''}>
          {Messages[item.id].map((paragraph) => (
            <Typography key={paragraph}>{paragraph}</Typography>
          ))}
        </TourStep>
      ),
    }));
};
