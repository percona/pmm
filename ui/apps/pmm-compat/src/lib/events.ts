/**
 * @fileoverview
 * Same events as in public/app/percona/shared/core/events.ts in grafana repository
 */

import { BusEventBase } from '@grafana/data';

export class SettingsUpdatedEvent extends BusEventBase {
  static type = 'settings-updated-event';
}

export class ServiceAddedEvent extends BusEventBase {
  static type = 'service-added-event';
}
