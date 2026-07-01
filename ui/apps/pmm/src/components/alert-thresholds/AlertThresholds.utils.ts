import { PrometheusAlertRulesResponse } from 'types/alerting.types';
import { AlertThresholdRow } from './AlertThresholds.types';

export const getInitialFormValues = (
  rows: AlertThresholdRow[]
): Record<string, number | undefined> =>
  rows.reduce(
    (acc, row) => {
      if (row.supportsOverride) {
        acc[row.ruleUid] = row.overrideThreshold || row.defaultThreshold;
      }
      return acc;
    },
    {} as Record<string, number | undefined>
  );

export const getNodeAlerts = (
  nodeName: string,
  response: PrometheusAlertRulesResponse
): AlertThresholdRow[] => {
  const groups: PrometheusAlertRulesResponse['data']['groups'] =
    response.data.groups;

  const rows: AlertThresholdRow[] = [];

  for (const group of groups) {
    for (const rule of group.rules) {
      const nodeAlert = getNodeAlert(nodeName, rule);

      if (nodeAlert && rule.query) {
        console.log('rule:', rule);

        rows.push({
          ruleUid: rule.uid || '',
          alertRuleName: rule.name,
          defaultThreshold: getDefaultThreshold(rule.query),
          overrideThreshold: getOverrideValue(nodeName, rule.query),
          supportsOverride: supportsOverride(rule.query),
        });
      }
    }
  }

  return rows;
};

const getNodeAlert = (
  nodeName: string,
  rule: PrometheusAlertRulesResponse['data']['groups'][0]['rules'][0]
) => {
  return rule.alerts.find((alert) => alert.labels?.node_name === nodeName);
};

const getDefaultThreshold = (
  query: PrometheusAlertRulesResponse['data']['groups'][0]['rules'][0]['query']
) => {
  if (!query) {
    return undefined;
  }

  if (query.includes('|')) {
    return getMultiPartDefaultValue(query);
  }

  return getQueryValue(query);
};

const getQueryValue = (query: string): number | undefined => {
  const value = query.split(' ').pop()?.trim() || undefined;
  return value ? parseFloat(value) : undefined;
};

const getMultiPartQueryDefaultValue = (
  nodeName: string,
  query: string
): number | undefined => {
  const parts = query.split('|').map((part) => part.trim());

  if (parts.length < 2) {
    return undefined;
  }

  return getNodeValue(parts[1], nodeName) ?? undefined;
};

const getOverrideValue = (
  nodeName: string,
  query: string
): number | undefined => {
  if (query.includes('|')) {
    return getMultiPartQueryDefaultValue(nodeName, query);
  }

  return undefined;
};

/**
 * 1. Given a node name, extract its vector() value.
 *    Matches:  label_replace(vector(10), "node_name", "<nodeName>", ...)
 *
 *    The number is captured BEFORE the labels because label_replace's
 *    signature is label_replace(v, dst_label, replacement, src_label, regex),
 *    so vector(10) comes first and "node_name"/"pmm-server" follow.
 */
const getNodeValue = (query: string, nodeName: string): number | undefined => {
  const node = nodeName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); // escape for regex
  const re = new RegExp(
    `label_replace\\(\\s*vector\\(\\s*(-?\\d+(?:\\.\\d+)?)\\s*\\)\\s*,\\s*"node_name"\\s*,\\s*"${node}"`
  );
  const m = query.match(re);
  return m ? parseFloat(m[1]) : undefined;
};

/**
 * 2. Extract the default value — the constant in the `... * 0 + <default>` idiom.
 *    The `\)` anchor before `* 0` avoids false positives from other arithmetic.
 */
const getMultiPartDefaultValue = (query: string): number | undefined => {
  const re = /\)\s*\*\s*0\s*\+\s*(-?\d+(?:\.\d+)?)/;
  const m = query.match(re);
  return m ? parseFloat(m[1]) : undefined;
};

const supportsOverride = (query: string): boolean =>
  query.includes('|') &&
  query.includes('label_replace') &&
  query.includes('* 0 +');
