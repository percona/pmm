// The case middleware only rewrites keys of plain objects, so a class instance passes through untouched.
class VerbatimKeys {}

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' &&
  value !== null &&
  Object.getPrototypeOf(value) === Object.prototype;

// Keeps the keys of user-defined maps (labels, DSN params) from being snake_cased on the way out.
export const withVerbatimKeys = <T>(data: T, mapFields: string[]): T => {
  if (!isPlainObject(data)) {
    return data;
  }

  return Object.fromEntries(
    Object.entries(data).map(([key, value]) => [
      key,
      mapFields.includes(key) && isPlainObject(value)
        ? Object.assign(new VerbatimKeys(), value)
        : withVerbatimKeys(value, mapFields),
    ])
  ) as T;
};
