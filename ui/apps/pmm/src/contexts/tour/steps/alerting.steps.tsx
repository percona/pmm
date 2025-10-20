import { StepType } from '@reactour/tour';
import { Messages } from './alerting.messages';
import { TourStep } from 'components/tour-step';
import Typography from '@mui/material/Typography';
import { User } from 'types/user.types';

export const getAlertingTourSteps = (user?: User): StepType[] => {
  const steps: StepType[] = [];

  steps.push({
    selector: '[data-testid="navitem-alerts-fired-list-item"]',
    content: (
      <TourStep title={Messages.firedAlerts.title}>
        <Typography>{Messages.firedAlerts.view}</Typography>
        <Typography>{Messages.firedAlerts.check}</Typography>
      </TourStep>
    ),
  });

  steps.push({
    selector: '[data-testid="navitem-alerts-rules-list-item"]',
    content: (
      <TourStep title={Messages.alertRules.title}>
        <Typography>{Messages.alertRules.rules}</Typography>
        <Typography>{Messages.alertRules.start}</Typography>
        <Typography>{Messages.alertRules.create}</Typography>
      </TourStep>
    ),
  });

  steps.push({
    selector: '[data-testid="navitem-alerts-contact-points-list-item"]',
    content: (
      <TourStep title={Messages.contactPoints.title}>
        <Typography>{Messages.contactPoints.define}</Typography>
        <Typography>{Messages.contactPoints.grafana}</Typography>
      </TourStep>
    ),
  });

  steps.push({
    selector: '[data-testid="navitem-alerts-policies-list-item"]',
    content: (
      <TourStep title={Messages.notificationPolicies.title}>
        <Typography>{Messages.notificationPolicies.routed}</Typography>
        <Typography>{Messages.notificationPolicies.policy}</Typography>
      </TourStep>
    ),
  });
  steps.push({
    selector: '[data-testid="navitem-alerts-silences-list-item"]',
    content: (
      <TourStep title={Messages.silences.title}>
        <Typography>{Messages.silences.create}</Typography>
        <Typography>{Messages.silences.silences}</Typography>
      </TourStep>
    ),
  });
  steps.push({
    selector: '[data-testid="navitem-alerts-groups-list-item"]',
    content: (
      <TourStep title={Messages.alertGroups.title}>
        <Typography>{Messages.alertGroups.alert}</Typography>
        <Typography>{Messages.alertGroups.grouping}</Typography>
      </TourStep>
    ),
  });

  if (user?.isPMMAdmin) {
    steps.push({
      selector: '[data-testid="navitem-alerts-settings-list-item"]',
      content: (
        <TourStep title={Messages.settings.title}>
          <Typography>{Messages.settings.configure}</Typography>
        </TourStep>
      ),
    });
  }

  if (user?.isEditor) {
    steps.push({
      selector: '[data-testid="navitem-alerts-templates-list-item"]',
      content: (
        <TourStep title={Messages.alertRuleTemplates.title}>
          <Typography>{Messages.alertRuleTemplates.effortlessly}</Typography>
          <Typography>{Messages.alertRuleTemplates.offers}</Typography>
        </TourStep>
      ),
    });
  }

  return steps;
};
