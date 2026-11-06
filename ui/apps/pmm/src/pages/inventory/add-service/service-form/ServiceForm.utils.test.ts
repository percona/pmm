import { InventoryNodeType } from 'types/nodes.types';
import { MetricsMode } from 'types/services.types';
import {
  getConnectionTimeoutError,
  getMetricsMode,
  getServiceName,
  isValidCollectors,
  isValidKeyValues,
  isValidPort,
  toCollectors,
  toKeyValueMap,
  toLabels,
  toOptionalNumber,
  toPlacement,
  whenTls,
} from './ServiceForm.utils';

describe('ServiceForm.utils', () => {
  it.each([
    ['3306', true],
    ['1', true],
    ['65534', true],
    ['0', false],
    ['22.5', false],
    ['65535', true],
    ['65536', false],
    ['', false],
    ['abc', false],
  ])('validates port %s', (port, valid) => {
    expect(isValidPort(port)).toBe(valid);
  });

  it.each([
    ['', true],
    ['env:prod', true],
    ['env:prod\nteam:db  owner:me', true],
    ['_private:1', true],
    ['env', false],
    ['env:', false],
    ['a:b:c', false],
    ['1env:prod', false],
    ['__reserved:x', false],
    ['my-label:x', false],
  ])('validates key-value text %j', (text, valid) => {
    expect(isValidKeyValues(text)).toBe(valid);
  });

  it.each([
    ['', true],
    ['  ', true],
    ['binlog_size, engine.innodb, info-schema', true],
    ['a,,b', true],
    ['bad collector', false],
    ['bad/collector', false],
  ])('validates collectors %j', (text, valid) => {
    expect(isValidCollectors(text)).toBe(valid);
  });

  it.each([
    ['', undefined],
    ['5s', undefined],
    ['1.5s', undefined],
    ['3600s', undefined],
    ['5', 'Invalid duration'],
    ['5m', 'Invalid unit. Allowed units: s'],
    ['500ms', 'Invalid unit. Allowed units: s'],
    ['-1s', 'Duration should be greater or equal to 0s'],
    ['3601s', 'Duration should be lower or equal to 3600s'],
  ])('validates connection timeout %j', (value, error) => {
    expect(getConnectionTimeoutError(value)).toBe(error);
  });

  it('turns key-value text into a map, keeping the keys as typed', () => {
    expect(toKeyValueMap('myLabel:a\nENV:prod other:x')).toEqual({
      myLabel: 'a',
      ENV: 'prod',
      other: 'x',
    });
    expect(toKeyValueMap(' \n')).toBeUndefined();
  });

  it('turns comma-separated collectors into a list', () => {
    expect(toCollectors(' a , b,,c ')).toEqual(['a', 'b', 'c']);
    expect(toCollectors('')).toBeUndefined();
  });

  it('reads optional numbers', () => {
    expect(toOptionalNumber('')).toBeUndefined();
    expect(toOptionalNumber('-1')).toBe(-1);
  });

  it('falls back to the address for the service name', () => {
    expect(getServiceName({ serviceName: ' ', address: ' db.local ' })).toBe(
      'db.local'
    );
    expect(getServiceName({ serviceName: 'mysql-1', address: 'db' })).toBe(
      'mysql-1'
    );
  });

  it('builds the label fields', () => {
    expect(
      toLabels({
        environment: 'prod',
        cluster: 'c1',
        replicationSet: 'rs',
        region: 'eu',
        az: 'a',
        customLabels: 'team:db',
      })
    ).toEqual({
      environment: 'prod',
      cluster: 'c1',
      replicationSet: 'rs',
      customLabels: { team: 'db' },
    });
  });

  it.each(['localhost', '127.0.0.1'])(
    'places a service on %s on the selected node',
    (address) => {
      expect(
        toPlacement(
          { address, nodeId: 'node-1', region: '', az: '' },
          'mysql-1'
        )
      ).toEqual({
        nodeId: 'node-1',
      });
    }
  );

  it('creates a remote node for any other address', () => {
    expect(
      toPlacement(
        {
          address: 'db.example.com',
          nodeId: 'n',
          region: 'eu-west-1',
          az: 'eu-west-1a',
        },
        'mysql-1'
      )
    ).toEqual({
      addNode: {
        nodeName: 'mysql-1',
        nodeType: InventoryNodeType.remote,
        region: 'eu-west-1',
        az: 'eu-west-1a',
      },
    });
  });

  it('pulls metrics on the PMM Server node and pushes them elsewhere', () => {
    expect(getMetricsMode(true)).toBe(MetricsMode.pull);
    expect(getMetricsMode(false)).toBe(MetricsMode.push);
  });

  it('sends TLS certificates only with TLS on', () => {
    const certificates = { tlsCa: 'ca', tlsCert: 'cert', tlsKey: 'key' };

    expect(whenTls(true, certificates)).toEqual(certificates);
    expect(whenTls(false, certificates)).toEqual({});
  });
});
