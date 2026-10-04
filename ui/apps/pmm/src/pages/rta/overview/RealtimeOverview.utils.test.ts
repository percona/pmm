import { describe, it, expect } from 'vitest';
import { ServiceType } from 'types/services.types';
import {
  TEST_REAL_TIME_SESSION,
  TEST_REAL_TIME_SESSION_MYSQL,
} from 'utils/testStubs';
import { RealtimeSessionStatus } from 'types/rta.types';
import {
  blockedOnlyTooltip,
  resolveSelection,
  sessionErrorsMessage,
} from './RealtimeOverview.utils';
import { Messages } from './RealtimeOverview.messages';

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

describe('sessionErrorsMessage', () => {
  const failed = {
    ...TEST_REAL_TIME_SESSION_MYSQL,
    status: RealtimeSessionStatus.error,
    statusMessage: 'Cannot connect to MySQL: connection refused',
  };

  it('names the service and the reason of a selected session that failed', () => {
    expect(
      sessionErrorsMessage([MYSQL_ID], [TEST_REAL_TIME_SESSION, failed])
    ).toBe(
      'Real-Time Analytics could not start for Service 3: Cannot connect to MySQL: connection refused'
    );
  });

  it('ignores sessions outside the selection', () => {
    expect(sessionErrorsMessage([MONGO_ID], [failed])).toBeUndefined();
  });

  it('ignores a running session that only carries warnings', () => {
    expect(
      sessionErrorsMessage(
        [MYSQL_ID],
        [{ ...failed, status: RealtimeSessionStatus.running }]
      )
    ).toBeUndefined();
  });

  it('keeps the default text for an error without a reason', () => {
    expect(
      sessionErrorsMessage(
        [MYSQL_ID],
        [{ ...failed, statusMessage: undefined }]
      )
    ).toBeUndefined();
  });
});

describe('blockedOnlyTooltip', () => {
  it('describes the plain filter when every row was judged', () => {
    expect(blockedOnlyTooltip(false, 0)).toBe(Messages.blockedOnlyTooltip);
  });

  it('points at the missing lock source when one could not be read', () => {
    const tooltip = blockedOnlyTooltip(true, 0);

    expect(tooltip).toContain('could not read every kind of lock');
    expect(tooltip).not.toContain('could not be attributed');
  });

  it('counts unattributed rows without blaming the lock sources', () => {
    const tooltip = blockedOnlyTooltip(false, 2);

    expect(tooltip).toContain('2 statements could not be attributed');
    expect(tooltip).toContain('moved on to a later statement');
    expect(tooltip).not.toContain('could not read every kind of lock');
  });

  it('uses the singular for one unattributed row', () => {
    expect(blockedOnlyTooltip(false, 1)).toContain(
      '1 statement could not be attributed this refresh'
    );
  });

  it('says both when both are true', () => {
    const tooltip = blockedOnlyTooltip(true, 3);

    expect(tooltip).toContain('could not read every kind of lock');
    expect(tooltip).toContain('3 statements could not be attributed');
  });
});
