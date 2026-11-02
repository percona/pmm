import { NAV_SECTIONS } from 'contexts/navigation/navigation.proposal';
import { NavItem } from 'types/navigation.types';

export interface StarredSplit {
  before: NavItem[];
  heading: NavItem;
  items: NavItem[];
  after: NavItem[];
}

// Finds the Starred heading and the starred entries that follow it, so the
// sidebar can render them as one section that can be edited.
export const splitStarred = (navTree: NavItem[]): StarredSplit | undefined => {
  const start = navTree.findIndex(
    (item) => item.id === NAV_SECTIONS.starred.id
  );

  if (start === -1) {
    return undefined;
  }

  let end = start + 1;

  while (end < navTree.length && navTree[end].pinned) {
    end++;
  }

  return {
    before: navTree.slice(0, start),
    heading: navTree[start],
    items: navTree.slice(start + 1, end),
    after: navTree.slice(end),
  };
};

// Applies a saved order. Entries it does not know yet keep their own order
// after the known ones.
export const orderItems = (items: NavItem[], order: string[]): NavItem[] => {
  const byId = new Map(items.map((item) => [item.id, item]));
  const known = order.flatMap((id) => byId.get(id) ?? []);

  return [...known, ...items.filter((item) => !order.includes(item.id))];
};

export const moveItem = <T>(list: T[], from: number, to: number): T[] => {
  const next = [...list];
  const [moved] = next.splice(from, 1);

  next.splice(to, 0, moved);

  return next;
};
