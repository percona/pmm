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

// One logfmt key=value pair; a quoted value may hold escaped quotes.
const LOGFMT_PAIR = /([A-Za-z_][\w.-]*)=("(?:[^"\\]|\\.)*"|[^\s"]*)/g;
// Keys the time and level columns already show, and the message itself.
const LOGFMT_SKIP = new Set(['time', 't', 'ts', 'level', 'lvl', 'msg']);

const unquote = (value: string): string => {
  if (!value.startsWith('"')) return value;
  try {
    return JSON.parse(value) as string;
  } catch {
    return value.slice(1, -1);
  }
};

/** A logfmt line's msg and its other fields, for lines that start with key=value and have a msg. */
export const logfmtParts = (
  line: LogLine
): { message: string; fields: string } | undefined => {
  const pairs = [...line.body.matchAll(LOGFMT_PAIR)];
  if (!pairs.length || pairs[0].index !== 0) return undefined;
  const msg = pairs.find((p) => p[1] === 'msg');
  if (!msg) return undefined;
  return {
    message: unquote(msg[2]),
    fields: pairs
      .filter((p) => !LOGFMT_SKIP.has(p[1]))
      .map((p) => p[0])
      .join(' '),
  };
};

/** Identifies a line across page loads, for the line the chart follows. */
export const lineKey = (line: LogLine): string =>
  [line.time, line.nodeName, line.fileName, line.patternHash].join('|');
