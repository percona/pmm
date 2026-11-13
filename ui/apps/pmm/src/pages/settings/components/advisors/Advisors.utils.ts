// splitEmailAddresses turns the recipients text field into the array the API takes, tolerating
// commas, semicolons and whitespace between entries.
export const splitEmailAddresses = (value: string): string[] =>
  value
    .split(/[;,\s]+/)
    .map((address) => address.trim())
    .filter(Boolean);
