/**
 * Formats date to "Month Day, Year, HH:MM UTC"
 * @param date
 * @returns formatted date
 */
export const formatCheckDate = (date: string) => {
  const dateObj = new Date(date);
  const month = dateObj.toLocaleDateString('en-US', { month: 'long' });
  const day = dateObj.getUTCDate();
  const year = dateObj.getUTCFullYear();
  const hours = String(dateObj.getUTCHours()).padStart(2, '0');
  const minutes = String(dateObj.getUTCMinutes()).padStart(2, '0');
  return `${month} ${day}, ${year}, ${hours}:${minutes} UTC`;
};
