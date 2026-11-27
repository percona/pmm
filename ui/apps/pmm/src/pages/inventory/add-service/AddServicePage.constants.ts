import type { FC } from 'react';
import { AddServiceType } from './AddServicePage.types';
import { AzureDiscovery } from './discovery/azure';
import { RdsDiscovery } from './discovery/rds';
import { ExternalForm } from './service-form/external';
import { HAProxyForm } from './service-form/haproxy';
import { MongoDBForm } from './service-form/mongodb';
import { MySQLForm } from './service-form/mysql';
import { PostgreSQLForm } from './service-form/postgresql';
import { ProxySQLForm } from './service-form/proxysql';
import { ValkeyForm } from './service-form/valkey';

export const SERVICE_FORMS: Record<AddServiceType, FC> = {
  [AddServiceType.rds]: RdsDiscovery,
  [AddServiceType.azure]: AzureDiscovery,
  [AddServiceType.mysql]: MySQLForm,
  [AddServiceType.mongodb]: MongoDBForm,
  [AddServiceType.postgresql]: PostgreSQLForm,
  [AddServiceType.proxysql]: ProxySQLForm,
  [AddServiceType.haproxy]: HAProxyForm,
  [AddServiceType.valkey]: ValkeyForm,
  [AddServiceType.external]: ExternalForm,
};
