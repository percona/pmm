import { describe, expect, it } from 'vitest';
import {
  AdvisorCheckTriggeredBy,
  type AdvisorRun,
  AdvisorRunStatus,
} from 'types/advisors.types';
import { triggeredByLabel, waitedForText } from './AdvisorRuns.utils';

const RUN: AdvisorRun = {
  id: 'run-late',
  triggeredBy: AdvisorCheckTriggeredBy.scheduler,
  status: AdvisorRunStatus.completed,
  startedAt: '2026-08-04T06:07:10Z',
  finishedAt: '2026-08-04T06:09:00Z',
  plannedChecksCount: 1,
  plannedServicesCount: 1,
  checksCount: 1,
  servicesCount: 1,
  findingsCount: 0,
  errorsCount: 0,
  severityCounts: [],
  checkNames: [],
  serviceIds: [],
  scheduleName: 'All checks daily',
  scheduledFor: '2026-08-04T06:00:00Z',
};

describe('triggeredByLabel', () => {
  it('names the schedule, and the user for Run now', () => {
    expect(triggeredByLabel(RUN)).toBe('All checks daily');
    expect(
      triggeredByLabel({ ...RUN, triggeredBy: AdvisorCheckTriggeredBy.user })
    ).toBe('User · All checks daily');
    expect(triggeredByLabel({ ...RUN, scheduleName: '' })).toBe('Scheduler');
  });
});

describe('waitedForText', () => {
  it('says what a late run waited for', () => {
    expect(
      waitedForText({
        ...RUN,
        waitedForRun: {
          runId: 'run-user',
          triggeredBy: AdvisorCheckTriggeredBy.user,
          startedAt: '2026-08-04T05:55:00Z',
        },
      })
    ).toBe(
      'Scheduled for 06:00 UTC. Waited 7 min for the run started at 05:55 UTC (a user run) to finish.'
    );
    expect(
      waitedForText({
        ...RUN,
        startedAt: '2026-08-04T07:05:00Z',
        waitedForRun: {
          runId: 'run-other',
          triggeredBy: AdvisorCheckTriggeredBy.scheduler,
          scheduleName: 'Prod security',
          startedAt: '2026-08-04T05:30:00Z',
        },
      })
    ).toBe(
      'Scheduled for 06:00 UTC. Waited 1 h 5 min for the run started at 05:30 UTC (Prod security) to finish.'
    );
  });

  it('says nothing for a run that started on time', () => {
    expect(waitedForText(RUN)).toBeNull();
  });
});
