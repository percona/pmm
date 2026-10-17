import { LogLine } from 'api/logs';
import { detailJson, previewText } from './logDisplay';

const line = (body: string, attr?: object): LogLine => ({
  time: '2026-10-17T10:30:00.000Z',
  nodeName: 'rs101',
  source: 'rs101',
  preset: 'mongodb',
  fileName: 'mongod.log',
  severityNumber: 9,
  severityText: 'I',
  body,
  bodyTruncated: false,
  attributes: attr ? { attr: JSON.stringify(attr) } : {},
});

describe('previewText', () => {
  it('joins a multi-line message into one line', () => {
    expect(
      previewText(line('statement: \n\t\t\tSELECT\n\t\t\t  datid,\n  datname'))
    ).toBe('statement: SELECT datid, datname');
  });

  it('adds what a MongoDB slow query ran on and how long it took', () => {
    expect(
      previewText(
        line('Slow query', {
          ns: 'shop.orders',
          type: 'command',
          command: { find: 'orders', $db: 'shop' },
          planSummary: 'COLLSCAN',
          docsExamined: 5000,
          nreturned: 3,
          durationMillis: 120,
        })
      )
    ).toBe(
      'Slow query · shop.orders · find · 120 ms · COLLSCAN · examined 5000 · returned 3'
    );
  });

  it('finds the command name among the other command fields', () => {
    // mongos forwards commands with their fields sorted, so the name is not the first key.
    expect(
      previewText(
        line('Slow query', {
          ns: 'shop.audit_log',
          command: {
            $audit: {},
            $db: 'shop',
            bypassDocumentValidation: false,
            documents: [],
            insert: 'audit_log',
          },
          durationMillis: 3,
        })
      )
    ).toBe('Slow query · shop.audit_log · insert · 3 ms');
  });

  it('names the replicated operation of an applied op', () => {
    expect(
      previewText(
        line('Applied op', {
          CRUD: { op: 'u', ns: 'config.transactions' },
          durationMillis: 1,
        })
      )
    ).toBe('Applied op · config.transactions · update · 1 ms');
  });

  it('lists the plain values of other MongoDB messages', () => {
    expect(
      previewText(
        line('Connection accepted', {
          remote: '127.0.0.1:53068',
          connectionId: 28948,
          connectionCount: 45,
        })
      )
    ).toBe(
      'Connection accepted · remote=127.0.0.1:53068 · connectionId=28948 · connectionCount=45'
    );
  });

  it('keeps the message when attr is not JSON', () => {
    expect(
      previewText({ ...line('Slow query'), attributes: { attr: '{oops' } })
    ).toBe('Slow query');
  });
});

describe('detailJson', () => {
  it('formats attr for reading', () => {
    expect(detailJson(line('Slow query', { ns: 'shop.orders' }))).toBe(
      '{\n  "ns": "shop.orders"\n}'
    );
  });

  it('is empty without attr', () => {
    expect(detailJson(line('statement: ;'))).toBe('');
  });
});
