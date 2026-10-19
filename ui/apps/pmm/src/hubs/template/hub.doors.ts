import { buildTableUrlParamKeys } from '@percona/peak-ui';
import { createRealtimeOverviewUrl } from 'utils/link.utils';
import { HUB_NO_ENVIRONMENT } from './hub.types';

export type DoorQueryValue = string | number | null | undefined;
export type DoorQuery = Record<string, DoorQueryValue | DoorQueryValue[]>;

/** How long before an incident's start its doors open the dashboards. */
export const INCIDENT_LEAD_MS = 30 * 60 * 1000;

export const QAN_DASHBOARD = 'pmm-qan/pmm-query-analytics';
export const ALERTS_PATH = '/alerting/status';
export const RTA_PATH = '/rta';

/** Alerts' column filter for the service, as `useTableUrlState` reads it (`?f.serviceName=`). */
export const ALERTS_SERVICE_FILTER_KEY = `${buildTableUrlParamKeys().filterPrefix}serviceName`;

const encode = (value: string | number) => encodeURIComponent(String(value));

/** Appends the query, skipping empty values; spaces encode as %20, which Grafana and React Router both read. */
export const withQuery = (path: string, query: DoorQuery = {}): string => {
  const pairs = Object.entries(query).flatMap(([key, value]) =>
    (Array.isArray(value) ? value : [value])
      .filter((item): item is string | number => item != null && item !== '')
      .map((item) => `${encode(key)}=${encode(item)}`)
  );

  if (!pairs.length) {
    return path;
  }

  return `${path}${path.includes('?') ? '&' : '?'}${pairs.join('&')}`;
};

/** A Grafana dashboard by `uid/slug`, with `var-` prefixed onto each variable. */
export const grafanaDashboard = (
  uidSlug: string,
  variables: DoorQuery = {},
  query: DoorQuery = {}
): string =>
  withQuery(`/graph/d/${uidSlug}`, {
    ...Object.fromEntries(
      Object.entries(variables).map(([name, value]) => [`var-${name}`, value])
    ),
    ...query,
  });

/** The environment a list door carries: the filter when one real environment is selected. */
export const listEnvironment = (filter: string | null): string | undefined =>
  filter && filter !== HUB_NO_ENVIRONMENT ? filter : undefined;

/** Frames a dashboard on an incident: from 30 minutes before it started to now. */
export const withIncidentFrame = (
  to: string,
  since: string | undefined
): string => {
  const start = since ? Date.parse(since) : NaN;

  return Number.isNaN(start)
    ? to
    : withQuery(to, { from: start - INCIDENT_LEAD_MS, to: 'now' });
};

export interface QanLinkOptions {
  serviceType?: string;
  environment?: string;
  cluster?: string;
  serviceName?: string;
  queryId?: string;
  from?: string;
}

/** Query analytics for a service type, optionally on one query with its details open. */
export const qanLink = ({
  serviceType,
  environment,
  cluster,
  serviceName,
  queryId,
  from,
}: QanLinkOptions = {}): string =>
  grafanaDashboard(
    QAN_DASHBOARD,
    {
      service_type: serviceType,
      environment,
      cluster,
      service_name: serviceName,
    },
    queryId
      ? {
          from: from ?? 'now-1h',
          to: 'now',
          filter_by: queryId,
          query_selected: 'true',
          details_tab: 'details',
        }
      : {}
  );

/** Alerts, filtered to one service when given. */
export const alertsLink = (serviceName?: string): string =>
  withQuery(ALERTS_PATH, { [ALERTS_SERVICE_FILTER_KEY]: serviceName });

/** Real-time queries for these services' sessions. */
export const realtimeLink = (serviceIds: string[] = []): string =>
  serviceIds.length ? createRealtimeOverviewUrl(serviceIds) : RTA_PATH;
