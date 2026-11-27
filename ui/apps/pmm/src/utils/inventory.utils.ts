import type { QueryClient } from '@tanstack/react-query';

// Removing an inventory entity can remove others (force mode) and changes the
// statuses and counts shown for the rest, so all inventory queries are refetched.
const INVENTORY_QUERY_KEYS = [
  ['services:managed'],
  ['services:editable'],
  ['services:list'],
  ['services:get'],
  ['services:getTypes'],
  ['nodes:list'],
  ['nodes:get'],
  ['agents:list'],
  ['agents:rdsExporter'],
];

export const invalidateInventory = (queryClient: QueryClient) =>
  Promise.all(
    INVENTORY_QUERY_KEYS.map((queryKey) =>
      queryClient.invalidateQueries({ queryKey })
    )
  );

// Resolves with how many removals succeeded; the API error notifications
// report the ones that failed.
export const removeEach = async <T>(
  items: T[],
  remove: (item: T) => Promise<void>
) => {
  const results = await Promise.allSettled(items.map((item) => remove(item)));
  return results.filter((result) => result.status === 'fulfilled').length;
};
