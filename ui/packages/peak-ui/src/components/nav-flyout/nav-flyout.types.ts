import type { ReactNode } from 'react';

export interface NavFlyoutProps {
  /** The panel's id, for the trigger's `aria-controls`. */
  id: string;
  /** The group's name: heads the panel and names its navigation landmark. */
  label: string;
  /** The item the panel opens beside, such as a group's icon in a collapsed rail. Open while set. */
  anchorEl: HTMLElement | null;
  /** Called on Escape or a click outside the panel. */
  onClose: () => void;
  /** Panel width; 250px by default. */
  width?: number | string;
  /** Moves focus into the panel as it opens, to the current page or else the first entry, and keeps it there. */
  autoFocus?: boolean;
  /** The group's entries: `NavItem`s, with any headings or actions between them. */
  children?: ReactNode;
}
