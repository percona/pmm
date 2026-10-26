import type { TechnologyHubConfig } from 'hubs/template/hub.types';
import type { MongoSnapshot } from '../mongodb.types';
import { emptySnapshot } from './empty';
import { forEnvironment } from './environment';
import {
  fleetHealthySnapshot,
  fleetIncompleteSnapshot,
  fleetUnhealthySnapshot,
} from './scenarios';

/** Raw snapshots for each prototype state, on the 60-cluster fleet, narrowed to the environment as live sources are. Their Needs attention rows are written out; the rest runs through the same derivation as live data. */
export const MONGODB_FIXTURES: TechnologyHubConfig<MongoSnapshot>['fixtures'] =
  {
    empty: (now, ops, environment) =>
      forEnvironment(emptySnapshot(now, ops), environment),
    healthy: (now, ops, environment) =>
      forEnvironment(fleetHealthySnapshot(now, ops), environment),
    incomplete: (now, ops, environment) =>
      forEnvironment(fleetIncompleteSnapshot(now, ops), environment),
    unhealthy: (now, ops, environment) =>
      forEnvironment(fleetUnhealthySnapshot(now, ops), environment),
  };
