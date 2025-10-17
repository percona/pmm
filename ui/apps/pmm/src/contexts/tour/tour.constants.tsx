import { StepType } from '@reactour/tour';
import { TourName } from './tour.context.types';
import { TourStep } from 'components/tour-step';
import { Link, Typography } from '@mui/material';

export const PRODUCT_TOUR_STEPS: StepType[] = [
  {
    selector: '[data-tour="system-list-item"]',
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
      <TourStep title="Percona Dashboards">
        <Typography>
          Here you'll find a collection of expertly designed dashboards, powered
          by Percona's deep database expertise, to monitor your databases
          seamlessly across on-premises, cloud, or hybrid environments.
        </Typography>
        <Typography>
          Built on Grafana, these dashboards can be customized, expanded, or
          integrated with existing ones to fit your unique monitoring needs.
        </Typography>
      </TourStep>
    ),
  },
  {
    selector: '[data-testid="navitem-qan-list-item"]',
    content: (
      <TourStep title="Query Analytics (QAN) dashboard">
        <Typography>
          Shows detailed insights into query execution, including query count
          and execution time.
        </Typography>
        <Typography>
          It enables you to analyze database queries over time, optimize
          performance, and quickly identify and resolve issues at their source.
        </Typography>
      </TourStep>
    ),
  },
  {
    selector: '[data-testid="navitem-explore-list-item"]',
    content: (
      <TourStep title="Explore">
        <Typography>
          Use it when you want to explore your graph and table data but do not
          want to create a dashboard. This way you can focus only on the query.
        </Typography>
      </TourStep>
    ),
  },
  {
    selector: '[data-testid="navitem-alerts-list-item"]',
    content: (
      <TourStep title="Alerts & Percona Templates">
        <Typography>
          Grafana’s built-in alerting is ready to use right out-of-the-box.
          Simply set metrics, define thresholds, and configure your
          communication channels to start receiving alerts.
        </Typography>
        <Typography>
          To speed up your workflow, use our pre-built alert templates developed
          based on our own expertise and real-world needs.
        </Typography>
        <Typography>
          <Link color="inherit">Read more about our templates</Link>
        </Typography>
      </TourStep>
    ),
  },
  // todo: recheck with Pedro
  {
    selector: '[data-testid="navitem-advisors-list-item"]',
    content: (
      <TourStep title="Percona Advisors">
        <Typography>
          Advisor checks proactively monitor your databases for potential
          security threats, performance degradation, data integrity risks,
          compliance issues, and more.
        </Typography>
        <Typography>
          <Link color="inherit">Read more about Advisors</Link>
        </Typography>
      </TourStep>
    ),
  },
  {
    selector: '[data-testid="navitem-inventory-list-item"]',
    highlightedSelectors: [
      '[data-testid="navitem-inventory-list-item"]',
      '[data-testid="navitem-backups-list-item"]',
    ],
    content: (
      <TourStep title="Management: Inventory & Backups">
        <Typography>
          Use the Inventory section to manage your monitored services, and the
          Backups section to create Point-in-Time-Recoverable backups of your
          MySQL and MongoDB databases.
        </Typography>
        <Typography>
          <Link color="inherit">Read more about Backups</Link>
        </Typography>
      </TourStep>
    ),
  },
  {
    selector: '[data-testid="navitem-configuration-list-item"]',
    highlightedSelectors: [
      '[data-testid="navitem-configuration-list-item"]',
      '[data-testid="navitem-users-and-access-list-item"]',
      '[data-testid="navitem-account-list-item"]',
    ],
    content: (
      <TourStep title="Configurations">
        <Typography>
          Manage PMM's advanced settings to customize metrics collection and
          control feature availability, including features in Technical Preview.
        </Typography>
        <Typography>
          Control team access and permissions through Users and access, or
          modify your personal preferences and password in your profile
          settings.
        </Typography>
        <Typography>
          <Link color="inherit">Learn more about PMM's configurations</Link>
        </Typography>
      </TourStep>
    ),
  },
  {
    selector: '[data-testid="navitem-help-list-item"]',
    highlightedSelectors: ['[data-testid="navitem-help-list-item"]'],
    content: (
      <TourStep title="Help Center">
        <Typography>
          Your one-stop resource for everything PMM. Whether you need
          documentation, community support, troubleshooting tools like datasets
          and logs, or useful tips to enhance your experience, it’s all just a
          click away.
        </Typography>
        <Typography>
          This PMM tour has finished but you can revisit it anytime from the
          Help Center too!
        </Typography>
      </TourStep>
    ),
  },
];

export const TOUR_STEPS_MAP: Record<TourName, StepType[]> = {
  product: PRODUCT_TOUR_STEPS,
};
