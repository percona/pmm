import { subHours, subMinutes } from 'date-fns';
import { type Dump, DumpStatus } from 'types/dump.types';

const now = new Date();
now.setSeconds(0, 0);

const iso = (date: Date) => date.toISOString();

export const MOCK_SERVICES = [
  'mysql-prod-primary',
  'mysql-prod-replica-1',
  'mysql-prod-replica-2',
  'mongo-rs0-node-1',
  'mongo-rs0-node-2',
  'mongo-rs0-node-3',
  'pg-orders-primary',
  'pg-orders-standby',
  'proxysql-edge',
  'valkey-cache',
];

export const MOCK_DUMPS: Dump[] = [
  {
    dumpId: '7f3c2a10-0b5e-4f0e-9a51-3b8f6a1c2d01',
    status: DumpStatus.InProgress,
    serviceNames: ['mysql-prod-primary', 'mysql-prod-replica-1'],
    startTime: iso(subHours(now, 6)),
    endTime: iso(now),
    createdAt: iso(subMinutes(now, 3)),
    encrypted: true,
  },
  {
    dumpId: '1b9e44d2-6c1a-4a3f-8e0d-7a2b9c5e4f12',
    status: DumpStatus.Success,
    serviceNames: [],
    startTime: iso(subHours(now, 14)),
    endTime: iso(subHours(now, 2)),
    createdAt: iso(subHours(now, 2)),
    encrypted: false,
  },
  {
    dumpId: 'c4d8e2f6-1a3b-4c5d-9e7f-0a1b2c3d4e23',
    status: DumpStatus.Error,
    serviceNames: ['mongo-rs0-node-1', 'mongo-rs0-node-2', 'mongo-rs0-node-3'],
    startTime: iso(subHours(now, 48)),
    endTime: iso(subHours(now, 24)),
    createdAt: iso(subHours(now, 23)),
    encrypted: false,
  },
  {
    dumpId: '9a0b1c2d-3e4f-4a5b-8c6d-7e8f9a0b1c34',
    status: DumpStatus.Success,
    serviceNames: [
      'pg-orders-primary',
      'pg-orders-standby',
      'proxysql-edge',
      'valkey-cache',
      'mysql-prod-primary',
    ],
    startTime: iso(subHours(now, 73)),
    endTime: iso(subHours(now, 72)),
    createdAt: iso(subHours(now, 70)),
    encrypted: true,
  },
];

export const MOCK_LOGS = [
  'time=2026-10-27T10:41:02Z level=info msg="Starting pmm-dump" version=0.8.0',
  'time=2026-10-27T10:41:02Z level=info msg="Checking PMM Server load" threshold_cpu=70 threshold_ram=80',
  'time=2026-10-27T10:41:03Z level=info msg="Exporting VictoriaMetrics data" services=2 chunks=72',
  'time=2026-10-27T10:41:09Z level=info msg="Processed 12/72 chunks"',
  'time=2026-10-27T10:41:16Z level=info msg="Processed 24/72 chunks"',
  'time=2026-10-27T10:41:24Z level=info msg="Processed 36/72 chunks"',
].join('\n');
