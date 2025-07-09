/**
 * Formats date to YYYY/MM/DD
 * @param date
 * @returns formatted date
 */
export const formatCheckDate = (date: string) =>
  new Date(date).toISOString().slice(0, 10).replace(/-/g, '/');
