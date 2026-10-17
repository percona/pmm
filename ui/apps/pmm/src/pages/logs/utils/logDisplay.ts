import { LogLine } from 'api/logs';

type JsonObject = Record<string, unknown>;

// MongoDB command names, so a command's name is found among its other fields.
const COMMANDS = new Set([
  'aggregate',
  'bulkWrite',
  'count',
  'create',
  'createIndexes',
  'delete',
  'distinct',
  'drop',
  'dropIndexes',
  'explain',
  'find',
  'findAndModify',
  'getMore',
  'hello',
  'insert',
  'isMaster',
  'killCursors',
  'listCollections',
  'listIndexes',
  'mapReduce',
  'ping',
  'update',
]);

// Oplog operation codes of a replicated write.
const OPLOG_OPS: Record<string, string> = {
  i: 'insert',
  u: 'update',
  d: 'delete',
  c: 'command',
  n: 'no-op',
};

const isObject = (v: unknown): v is JsonObject =>
  !!v && typeof v === 'object' && !Array.isArray(v);

/** MongoDB's attr field, the details of a JSON log line, when the line has one. */
const parseAttr = (line: LogLine): JsonObject | undefined => {
  const raw = line.attributes.attr;
  if (!raw) return undefined;
  try {
    const value: unknown = JSON.parse(raw);
    return isObject(value) ? value : undefined;
  } catch {
    return undefined;
  }
};

/** What ran, on what, how long and how; the plain values when attr names no operation. */
const attrSummary = (attr: JsonObject): string[] => {
  const crud = isObject(attr.CRUD) ? attr.CRUD : undefined;
  const command = isObject(attr.command) ? attr.command : undefined;
  const op = crud
    ? (OPLOG_OPS[String(crud.op)] ?? crud.op)
    : command && Object.keys(command).find((k) => COMMANDS.has(k));
  const known = [
    attr.ns ?? crud?.ns,
    op,
    attr.durationMillis !== undefined && `${attr.durationMillis} ms`,
    attr.planSummary,
    attr.docsExamined !== undefined && `examined ${attr.docsExamined}`,
    attr.nreturned !== undefined && `returned ${attr.nreturned}`,
  ].filter((v) => v !== undefined && v !== false && v !== '');
  if (known.length) return known.map(String);
  return Object.entries(attr)
    .filter(([, v]) => ['string', 'number', 'boolean'].includes(typeof v))
    .map(([k, v]) => `${k}=${v}`);
};

/** One line for the list: the message on a single line, plus a summary of MongoDB's attr. */
export const previewText = (line: LogLine): string => {
  const text = line.body.replace(/\s+/g, ' ').trim();
  const attr = parseAttr(line);
  return attr ? [text, ...attrSummary(attr)].join(' · ') : text;
};

/** MongoDB's attr formatted for reading; empty when the line has none. */
export const detailJson = (line: LogLine): string => {
  const attr = parseAttr(line);
  return attr ? JSON.stringify(attr, null, 2) : '';
};
