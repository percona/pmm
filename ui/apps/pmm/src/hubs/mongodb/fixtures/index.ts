import type { TechnologyHubConfig } from 'hubs/template/hub.types';
import type { MongoSnapshot } from '../mongodb.types';
import { emptySnapshot } from './empty';
import { forEnvironment } from './environment';
import { healthySnapshot } from './healthy';
import { wrongSnapshot } from './wrong';

/** Raw snapshots for each prototype state, narrowed to the environment as live sources are; they run through the same derivation as live data. */
export const MONGODB_FIXTURES: TechnologyHubConfig<MongoSnapshot>['fixtures'] =
  {
    empty: (now, ops, environment) =>
      forEnvironment(emptySnapshot(now, ops), environment),
    healthy: (now, ops, environment) =>
      forEnvironment(healthySnapshot(now, ops), environment),
    wrong: (now, ops, environment) =>
      forEnvironment(wrongSnapshot(now, ops), environment),
  };
