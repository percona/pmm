// material-react-table hands the filter value back untyped, and a range can
// arrive half filled or, from the URL, not as an array at all.
export const readRange = (value: unknown): [string, string] =>
  Array.isArray(value)
    ? [String(value[0] ?? ''), String(value[1] ?? '')]
    : ['', ''];
