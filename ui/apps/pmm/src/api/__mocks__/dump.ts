import { DumpStatus } from 'types/dump.types';

export const listDumps = vi.fn().mockResolvedValue({
  dumps: [
    {
      dumpId: 'dump-1',
      status: DumpStatus.Success,
      serviceNames: ['mysql'],
      startTime: '2026-07-10T08:00:00Z',
      endTime: '2026-07-10T10:00:00Z',
      createdAt: '2026-07-10T10:01:00Z',
      encrypted: false,
    },
  ],
});

export const startDump = vi.fn().mockResolvedValue({ dumpId: 'dump-2' });
export const deleteDumps = vi.fn().mockResolvedValue({});
export const uploadDumps = vi.fn().mockResolvedValue({});
export const getDumpLogs = vi.fn().mockResolvedValue({
  logs: [{ chunkId: 1, data: 'dump log' }],
  end: true,
});
