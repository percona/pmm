import type { TechnologyHubConfig } from 'hubs/template/hub.types';
import type { MongoSnapshot } from '../mongodb.types';
import { emptySnapshot } from './empty';
import { healthySnapshot } from './healthy';
import { wrongSnapshot } from './wrong';

/** Raw snapshots for each prototype state; they run through the same derivation as live data. */
export const MONGODB_FIXTURES: TechnologyHubConfig<MongoSnapshot>['fixtures'] =
  {
    empty: emptySnapshot,
    healthy: healthySnapshot,
    wrong: wrongSnapshot,
  };
