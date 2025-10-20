import { StepType } from '@reactour/tour';
import { Messages } from './product.messages';
import { TourStep } from 'components/tour-step';
import { Link, Typography } from '@mui/material';
import { User } from 'types/user.types';

export const getProductTourSteps = (user?: User): StepType[] => {
  const steps: StepType[] = [
    {
      selector: '[data-testid="navitem-system-list-item"]',
      highlightedSelectors: [
        '[data-testid="navitem-mysql-list-item"]',
        '[data-testid="navitem-mongo-list-item"]',
        '[data-testid="navitem-postgre-list-item"]',
        '[data-testid="navitem-system-list-item"]',
        '[data-testid="navitem-haproxy-list-item"]',
        '[data-testid="navitem-proxysql-list-item"]',
        '[data-testid="navitem-dashboards-list-item"]',
      ],
      content: (
        <TourStep title={Messages.perconaDashboards.title}>
          <Typography>{Messages.perconaDashboards.collection}</Typography>
          <Typography>{Messages.perconaDashboards.customization}</Typography>
        </TourStep>
      ),
    },
    {
      selector: '[data-testid="navitem-qan-list-item"]',
      content: (
        <TourStep title={Messages.queryAnalytics.title}>
          <Typography>{Messages.queryAnalytics.insights}</Typography>
          <Typography>{Messages.queryAnalytics.optimize}</Typography>
        </TourStep>
      ),
    },
  ];

  if (user?.isEditor) {
    steps.push({
      selector: '[data-testid="navitem-explore-list-item"]',
      content: (
        <TourStep title={Messages.explore.title}>
          <Typography>{Messages.explore.useCase}</Typography>
        </TourStep>
      ),
    });
    steps.push({
      selector: '[data-testid="navitem-alerts-list-item"]',
      content: (
        <TourStep title={Messages.alerts.title}>
          <Typography>{Messages.alerts.builtin}</Typography>
          <Typography>{Messages.alerts.templates}</Typography>
          <Typography>
            <Link color="inherit">{Messages.alerts.readMore}</Link>
          </Typography>
        </TourStep>
      ),
    });
    steps.push({
      selector: '[data-testid="navitem-advisors-list-item"]',
      content: (
        <TourStep title={Messages.advisors.title}>
          <Typography>{Messages.advisors.checks}</Typography>
          <Typography>
            <Link color="inherit">{Messages.advisors.readMore}</Link>
          </Typography>
        </TourStep>
      ),
    });
  }

  if (user?.isPMMAdmin) {
    steps.push({
      selector: '[data-testid="navitem-inventory-list-item"]',
      highlightedSelectors: [
        '[data-testid="navitem-inventory-list-item"]',
        '[data-testid="navitem-backups-list-item"]',
      ],
      content: (
        <TourStep title={Messages.management.title}>
          <Typography>{Messages.management.inventory}</Typography>
          <Typography>
            <Link color="inherit">{Messages.management.readMore}</Link>
          </Typography>
        </TourStep>
      ),
    });
    steps.push({
      selector: '[data-testid="navitem-configuration-list-item"]',
      highlightedSelectors: [
        '[data-testid="navitem-configuration-list-item"]',
        '[data-testid="navitem-users-and-access-list-item"]',
        '[data-testid="navitem-account-list-item"]',
      ],
      content: (
        <TourStep title={Messages.configurations.title}>
          <Typography>{Messages.configurations.settings}</Typography>
          <Typography>{Messages.configurations.access}</Typography>
          <Typography>
            <Link color="inherit">{Messages.configurations.readMore}</Link>
          </Typography>
        </TourStep>
      ),
    });
  } else {
    steps.push({
      selector: '[data-testid="navitem-account-list-item"]',
      content: (
        <TourStep title={Messages.account.title}>
          <Typography>{Messages.account.personalPreferences}</Typography>
        </TourStep>
      ),
    });
  }

  steps.push({
    selector: '[data-testid="navitem-help-list-item"]',
    highlightedSelectors: ['[data-testid="navitem-help-list-item"]'],
    content: (
      <TourStep title={Messages.helpCenter.title}>
        <Typography>{Messages.helpCenter.resource}</Typography>
        <Typography>{Messages.helpCenter.revisit}</Typography>
      </TourStep>
    ),
  });

  return steps;
};
