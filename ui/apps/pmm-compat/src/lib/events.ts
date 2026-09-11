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

export class ServiceDeletedEvent extends BusEventBase {
  static type = 'service-deleted-event';
}

export class FrontendSettingsUpdatedEvent extends BusEventBase {
  static type = 'frontend-settings-updated-event';
}

export class TimeZoneUpdatedEvent extends BusEventBase {
  static type = 'timezone-updated-event';
}

export class OpenAlertThresholdsModalEvent extends BusEventBase {
  static type = 'open-alert-thresholds-modal-event';
}
