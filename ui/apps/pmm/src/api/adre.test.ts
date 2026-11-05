import { describe, expect, it } from 'vitest';
import { getAlertMetadataFromLabels } from './adre';

describe('getAlertMetadataFromLabels', () => {
  it('reads node and service names from PMM alert labels', () => {
    expect(
      getAlertMetadataFromLabels({
        node_name: 'pdpgsql_pmm_17_1',
        service_name: 'pdpgsql_pmm_17_1_5975',
        severity: 'warning',
      })
    ).toEqual({
      nodeName: 'pdpgsql_pmm_17_1',
      serviceName: 'pdpgsql_pmm_17_1_5975',
      clusterName: undefined,
      severity: 'warning',
    });
  });

  it('treats the Grafana [no value] placeholder as a missing label', () => {
    expect(
      getAlertMetadataFromLabels({
        node_name: '[no value]',
        service_name: '[no value]',
        set: 'rs2',
      })
    ).toEqual({
      nodeName: undefined,
      serviceName: undefined,
      clusterName: undefined,
      severity: undefined,
    });
  });
});
