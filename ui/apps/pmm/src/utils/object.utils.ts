// Only object literals and null-prototype objects qualify. Dates, Maps, Sets,
// arrays and class instances are excluded: their data does not live in own
// enumerable properties, so treating them as plain would lose it.
export const isPlainObject = (
  value: unknown
): value is Record<string, unknown> => {
  if (typeof value !== 'object' || value === null) {
    return false;
  }

  const prototype = Object.getPrototypeOf(value);

  return prototype === Object.prototype || prototype === null;
};
