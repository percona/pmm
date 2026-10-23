import type { NavItem } from 'types/navigation.types';

export type SearchItemKind =
  | 'page'
  | 'dashboard'
  | 'folder'
  | 'setting'
  | 'action'
  | 'help'
  | 'service'
  | 'node'
  | 'link'
  | 'retry';

export type SearchGroupId =
  | 'recent'
  | 'pages'
  | 'settings'
  | 'actions'
  | 'dashboards'
  | 'inventory'
  | 'help';

// Structurally a Peak UI SearchDialogItem without `icon` and `meta`, so the pure modules never import Peak UI.
export interface SearchItem {
  id: string;
  kind: SearchItemKind;
  group: SearchGroupId;
  label: string;
  description?: string | string[];
  keywords: string[];
  // Other names of the item's technology: they match as words, never as the whole query.
  aliases?: string[];
  navIcon?: NavItem['icon'];
  external?: boolean;
  status?: 'error';
  // A router path, which may carry a query.
  to?: string;
  // Outside the router: external URLs, /logs.zip, /graph/logout and /graph/login.
  href?: string;
  target?: '_self' | '_blank';
  perform?: () => void;
  // `to` is a /graph/d/… path with no query.
  carryVariables?: boolean;
  starred?: boolean;
  curated?: boolean;
  // From /graph/api/search or the inventory, so it needs at least two characters.
  remote?: boolean;
  dedupeKey: string;
  // Position in the merged index; the last tie-break.
  order: number;
}

export interface SearchResultGroup {
  id: SearchGroupId;
  label: string;
  items: SearchItem[];
}

export interface RecentEntry {
  id: string;
  kind: SearchItemKind;
  label: string;
  description?: string | string[];
  to: string;
  at: number;
}

export interface RemoteState {
  status: 'idle' | 'loading' | 'error' | 'ready';
  refetch: () => void;
}

// Peak UI's SearchDialogSelectDetails is assignable to this.
export interface SelectDetails {
  newTab: boolean;
}

export interface SearchIndex {
  items: SearchItem[];
  staticItems: SearchItem[];
  dashboards: RemoteState;
  inventory: RemoteState;
}
