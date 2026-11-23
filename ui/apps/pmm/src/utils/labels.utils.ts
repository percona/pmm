// Prometheus label names, see https://prometheus.io/docs/concepts/data_model
const LABEL_NAME_RE = /^[a-zA-Z_][a-zA-Z0-9_]*$/;

export const toCustomLabelsText = (labels: Record<string, string>) =>
  Object.entries(labels)
    .map(([name, value]) => `${name}:${value}`)
    .join('\n');

// Only label names are restricted, so a value can hold colons too
export const splitCustomLabel = (entry: string): [string, string] => {
  const separator = entry.indexOf(':');

  return separator === -1
    ? [entry, '']
    : [entry.slice(0, separator), entry.slice(separator + 1).trim()];
};

// Labels are separated by new lines or spaces, but a space only starts a new
// label before another "name:", so values can hold spaces
export const splitCustomLabelEntries = (text: string) =>
  text
    .split(/\n|\s+(?=[^\s:]+:)/)
    .map((entry) => entry.trim())
    .filter(Boolean);

export const parseCustomLabels = (text: string): Record<string, string> =>
  Object.fromEntries(splitCustomLabelEntries(text).map(splitCustomLabel));

export const areCustomLabelsValid = (text: string) =>
  splitCustomLabelEntries(text).every((entry) => {
    const [name, value] = splitCustomLabel(entry);

    return LABEL_NAME_RE.test(name) && !name.startsWith('__') && !!value;
  });
