import { describe, it, expect } from 'vitest';
import { ServiceType } from 'types/services.types';
import {
  TEST_REAL_TIME_SESSION,
  TEST_REAL_TIME_SESSION_MYSQL,
} from 'utils/testStubs';
import { resolveSelection } from './RealtimeOverview.utils';

const MONGO_ID = TEST_REAL_TIME_SESSION.serviceId;
const MYSQL_ID = TEST_REAL_TIME_SESSION_MYSQL.serviceId;
const SESSIONS = [TEST_REAL_TIME_SESSION, TEST_REAL_TIME_SESSION_MYSQL];

describe('resolveSelection', () => {
  it('reports the technology of a single-technology selection', () => {
    expect(resolveSelection([MONGO_ID], SESSIONS)).toEqual({
      serviceIds: [MONGO_ID],
      serviceType: ServiceType.mongodb,
    });
  });

  it('keeps only the first technology when a URL names both', () => {
    expect(resolveSelection([MYSQL_ID, MONGO_ID], SESSIONS)).toEqual({
      serviceIds: [MYSQL_ID],
      serviceType: ServiceType.mysql,
    });

    // Order decides which one wins, so the same pair the other way round keeps
    // the MongoDB service.
    expect(resolveSelection([MONGO_ID, MYSQL_ID], SESSIONS)).toEqual({
      serviceIds: [MONGO_ID],
      serviceType: ServiceType.mongodb,
    });
  });

  it('passes the selection through before the sessions have loaded', () => {
    expect(resolveSelection([MYSQL_ID, MONGO_ID], [])).toEqual({
      serviceIds: [MYSQL_ID, MONGO_ID],
    });
  });

  it('keeps services that have no running session', () => {
    expect(resolveSelection([MYSQL_ID, 'no-session'], SESSIONS)).toEqual({
      serviceIds: [MYSQL_ID, 'no-session'],
      serviceType: ServiceType.mysql,
    });
  });

  it('returns an empty selection unchanged', () => {
    expect(resolveSelection([], SESSIONS)).toEqual({ serviceIds: [] });
  });
});
