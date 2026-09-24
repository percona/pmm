import { GRAFANA_SUB_PATH } from 'lib/constants';

export function makeLabelBasedSilenceLink(
  alertManagerSourceName: string,
  labels: Record<string, string>
) {
  const silenceUrlParams = new URLSearchParams();
  silenceUrlParams.append('alertmanager', alertManagerSourceName);

  const matcherParams = getMatcherQueryParams(labels);
  matcherParams.forEach((value, key) => silenceUrlParams.append(key, value));

  return createRelativeUrl('/alerting/silence/new', silenceUrlParams);
}

export const getMatcherQueryParams = (labels: Record<string, string>) => {
  const validMatcherLabels = Object.entries(labels).filter(
    ([labelKey]) => !isPrivateLabelKey(labelKey)
  );

  const matcherUrlParams = new URLSearchParams();
  validMatcherLabels.forEach(([labelKey, labelValue]) =>
    matcherUrlParams.append('matcher', `${labelKey}=${labelValue}`)
  );

  return matcherUrlParams;
};

export function isPrivateLabelKey(labelKey: string) {
  return (
    (labelKey.startsWith('__') && labelKey.endsWith('__')) ||
    labelKey === GRAFANA_ORIGIN_LABEL
  );
}

export function createRelativeUrl(
  path: string,
  queryParams?: string[][] | Record<string, string> | string | URLSearchParams
) {
  const searchParams = new URLSearchParams(queryParams);
  const searchParamsString = searchParams.toString();

  return `${GRAFANA_SUB_PATH}${path}${searchParamsString ? `?${searchParamsString}` : ''}`;
}

export const GRAFANA_ORIGIN_LABEL = '__grafana_origin';
