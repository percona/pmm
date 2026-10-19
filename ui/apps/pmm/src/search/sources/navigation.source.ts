import { NAV_ACCOUNT } from 'contexts/navigation/navigation.constants';
import { NAV_SECTIONS } from 'contexts/navigation/navigation.proposal';
import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';
import type { NavItem } from 'types/navigation.types';
import { Messages } from '../search.messages';
import { searchItem, TECH_ALIASES, TECHNOLOGIES } from '../search.keywords';
import { canonicalKey, carriesVariables, techLabel } from '../search.text';
import type { SearchItem } from '../search.types';

export interface NavSourceContext {
  isAdmin: boolean;
}

export interface FlattenedNav {
  pages: SearchItem[];
  actions: SearchItem[];
  starredDashboards: SearchItem[];
}

const isDashboardKey = (key?: string): key is string =>
  !!key?.startsWith('dashboard:');

const isDashboardUrl = (url?: string) => isDashboardKey(canonicalKey(url));

// Grafana redirects its root to the home dashboard, which addHomePage lists last in the entry's matches.
const homeDashboardKey = (item: NavItem) =>
  canonicalKey(item.url) === PMM_NEW_NAV_GRAFANA_PATH
    ? item.matches
        ?.map((url) => canonicalKey(url))
        .filter(isDashboardKey)
        .pop()
    : undefined;

// The Account entry carries the user's first name after a colon, which search never shows.
const navText = (item: NavItem) => {
  const text = item.text ?? '';

  return item.id === NAV_ACCOUNT.id ? text.split(':')[0] : text;
};

// A group that opens the same page as one of its children gives way to that child.
const sharedChild = (item: NavItem) => {
  const key = canonicalKey(item.url);

  return key
    ? item.children?.find(
        (child) => !child.hidden && canonicalKey(child.url) === key
      )
    : undefined;
};

const groupKeywords = (item: NavItem) => {
  const shared = sharedChild(item);

  return new Map<NavItem, string[]>(shared ? [[shared, [navText(item)]]] : []);
};

export const flattenNavTree = (
  navTree: NavItem[],
  { isAdmin }: NavSourceContext
): FlattenedNav => {
  const pages: SearchItem[] = [];
  const actions: SearchItem[] = [];
  const starredDashboards: SearchItem[] = [];

  const walk = (
    items: NavItem[],
    crumbs: string[],
    rootIcon: NavItem['icon'],
    inheritedKeywords: Map<NavItem, string[]>,
    topLevel: boolean
  ) => {
    let section: string | undefined;
    let inStarred = false;

    items.forEach((item) => {
      if (
        item.hidden ||
        item.type === 'menu-divider' ||
        item.type === 'menu-text'
      ) {
        return;
      }

      if (item.type === 'menu-section') {
        section = item.text;
        inStarred = topLevel && item.id === NAV_SECTIONS.starred.id;
        return;
      }

      const text = navText(item);
      const path = section ? [...crumbs, section] : crumbs;
      const description = path.length ? path : undefined;
      const icon = item.icon ?? rootIcon;
      const keywords = inheritedKeywords.get(item) ?? [];
      const hasChildren = !!item.children?.length;

      if (item.action && isAdmin) {
        actions.push(
          searchItem({
            id: `action:${item.id}-action`,
            kind: 'action',
            group: 'actions',
            label: item.action.label,
            description: [...path, text],
            navIcon: item.action.icon,
            to: item.action.url,
          })
        );
      }

      if (item.url && item.target) {
        actions.push(
          searchItem({
            id: `action:${item.id}`,
            kind: 'action',
            group: 'actions',
            label: text,
            description,
            keywords,
            navIcon: item.icon,
            href: item.url,
            target: item.target === '_self' ? '_self' : '_blank',
            perform: item.onClick,
          })
        );
      } else if (item.url && !(hasChildren && sharedChild(item))) {
        const fields = {
          label: text,
          description,
          keywords,
          navIcon: icon,
          to: item.url,
          carryVariables: carriesVariables(item.url),
        };

        if (inStarred && isDashboardUrl(item.url)) {
          starredDashboards.push(
            searchItem({
              ...fields,
              id: canonicalKey(item.url) ?? `page:${item.id}`,
              kind: 'dashboard',
              group: 'dashboards',
              starred: true,
            })
          );
        } else {
          pages.push(
            searchItem({
              ...fields,
              id: `page:${item.id}`,
              kind: 'page',
              group: 'pages',
              starred: item.pinned === true,
              dedupeKey: homeDashboardKey(item),
            })
          );
        }
      } else if (!item.url && item.onClick) {
        actions.push(
          searchItem({
            id: `action:${item.id}`,
            kind: 'action',
            group: 'actions',
            label: text,
            description,
            keywords,
            navIcon: item.icon,
            perform: item.onClick,
          })
        );
      }

      if (hasChildren) {
        walk(
          item.children ?? [],
          [...path, text],
          topLevel ? icon : rootIcon,
          groupKeywords(item),
          false
        );
      }
    });
  };

  walk(navTree, [], undefined, new Map(), true);

  return { pages, actions, starredDashboards };
};

// The dashboards behind each technology entry, which the proposed sidebar no longer lists, except one the entry itself opens.
export const technologyDashboards = (navTree: NavItem[]): SearchItem[] =>
  TECHNOLOGIES.flatMap((tech) => {
    const techText = tech.text ?? '';
    const techKey = canonicalKey(
      navTree.find((item) => item.id === tech.id)?.url
    );
    const aliases = TECH_ALIASES[tech.id] ?? [];
    const dashboards: SearchItem[] = [];

    const walk = (
      children: NavItem[],
      parents: string[],
      inheritedKeywords: Map<NavItem, string[]>
    ) =>
      children.forEach((child) => {
        const text = child.text ?? '';
        const key = canonicalKey(child.url);
        const givesWay = !!child.children?.length && !!sharedChild(child);

        if (child.url && key && key !== techKey && !givesWay) {
          dashboards.push(
            searchItem({
              id: key,
              kind: 'dashboard',
              group: 'dashboards',
              label: techLabel(techText, text),
              description: [Messages.crumbs.technologies, techText, ...parents],
              keywords: [text, ...(inheritedKeywords.get(child) ?? [])],
              aliases,
              navIcon: child.icon ?? tech.icon,
              to: child.url,
              carryVariables: carriesVariables(child.url),
              curated: true,
            })
          );
        }

        if (child.children?.length) {
          walk(child.children, [...parents, text], groupKeywords(child));
        }
      });

    walk(tech.children ?? [], [], new Map());

    return dashboards;
  });
