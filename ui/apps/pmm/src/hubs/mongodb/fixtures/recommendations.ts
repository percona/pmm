import type { HubRecommendation } from 'hubs/template/hub.types';
import { MongoDestinations as To } from '../mongodb.destinations';
import { Messages } from '../mongodb.messages';

// The Recommendations rows of each demo scenario, written in the format the requirements use; each list is in display order.

const S = Messages.sources;
const ALL = Messages.recommendations.allClusters;

/** Behind on patches, which Operations' version check knows; without Operations, what it would take to know. */
const patches = (operations: boolean): HubRecommendation =>
  operations
    ? {
        kind: 'info',
        title: '14 clusters are behind on patches',
        impact: 'Newer patch releases already run elsewhere in your fleet.',
        location: 'billing-us, catalog-eu, payments-apac and 11 more',
        clusterCount: 14,
        source: S.versions,
        destination: To.planInFleet(),
      }
    : {
        kind: 'notset',
        title: 'Patch levels aren’t tracked',
        impact:
          'Turn on Operations for MongoDB to see which clusters are behind on patches.',
        location: ALL,
        clusterCount: 60,
        source: S.versions,
        destination: To.turnOnOperations(),
      };

const PITR: HubRecommendation = {
  kind: 'info',
  title: 'Point-in-time recovery is off on 9 production clusters',
  impact: 'They can only be restored to their last scheduled backup.',
  location: 'orders-us, cart-us, media-eu and 6 more',
  environment: 'prod',
  clusterCount: 9,
  source: S.backups,
  destination: To.turnOnPitr(),
};

const QAN: HubRecommendation = {
  kind: 'info',
  title: 'Query analytics can’t see queries on 12 clusters',
  impact: 'Query collection isn’t enabled, so slow queries go unnoticed.',
  location: 'search-eu, events-us, media-apac and 9 more',
  clusterCount: 12,
  source: S.qan,
  destination: To.howToEnableQan(),
};

const SECURITY: HubRecommendation = {
  kind: 'info',
  title: 'Advisors flagged 7 clusters for security settings',
  impact: 'For example, authentication is disabled on 3 development clusters.',
  location: 'sandbox-dev, analytics-dev and 5 more',
  clusterCount: 7,
  source: S.advisors,
  destination: To.advisors(),
};

// From the versions the members report, so it stays whether Operations is on or off.
const V60: HubRecommendation = {
  kind: 'info',
  title: '6 clusters run MongoDB 6.0',
  impact: 'The rest of your fleet runs 7.0 or 8.0.',
  location: 'reports-eu, audit-us and 4 more',
  clusterCount: 6,
  source: S.health,
  destination: To.upgradeGuide(),
};

const LAG_RULES: HubRecommendation = {
  kind: 'info',
  title: 'Replication lag and oplog window have no alert rules',
  impact: 'You’ll only see these problems on this page, not as a notification.',
  location: ALL,
  clusterCount: 60,
  source: S.alerts,
  destination: To.alertTemplates(),
};

const noBackups = (count: number, location: string): HubRecommendation => ({
  kind: 'notset',
  title: `${count} clusters have no backups`,
  impact: 'If data is lost, there’s nothing to restore from.',
  location,
  clusterCount: count,
  source: S.backups,
  destination: To.setUpBackups(),
});

/** No backup gap: the banner says backups are on schedule. */
export const healthyRecommendations = (
  operations: boolean
): HubRecommendation[] => [
  PITR,
  SECURITY,
  patches(operations),
  V60,
  QAN,
  LAG_RULES,
];

export const incompleteRecommendations = (
  operations: boolean
): HubRecommendation[] => [
  {
    kind: 'notset',
    title: 'No alert rules cover MongoDB',
    impact: 'PMM won’t notify anyone if a cluster goes down.',
    location: ALL,
    clusterCount: 60,
    source: S.alerts,
    destination: To.alertTemplates(),
  },
  noBackups(19, 'reports-eu, users-apac, sandbox-dev and 16 more'),
  PITR,
  patches(operations),
  V60,
  QAN,
];

export const unhealthyRecommendations = (
  operations: boolean
): HubRecommendation[] => [
  noBackups(4, 'reports-eu, sandbox-dev and 2 more'),
  PITR,
  SECURITY,
  patches(operations),
  V60,
  QAN,
];
