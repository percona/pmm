import type { BoxProps } from '@mui/material/Box';
import type { IconButtonProps } from '@mui/material/IconButton';
import type { PaperProps } from '@mui/material/Paper';
import type { SlideProps } from '@mui/material/Slide';
import type { TabProps } from '@mui/material/Tab';
import type { ReactNode } from 'react';
import type { UseDetailsPaneNavigationResult } from '../table/useDetailsPaneNavigation';

export interface DetailsPaneTab {
  /** Identifies the tab; `tab`, `defaultTab` and `onTabChange` refer to it. */
  value: string;
  /** Tab title the user clicks. */
  label: ReactNode;
  /** What the pane shows while this tab is selected. */
  content: ReactNode;
  /** Props for the MUI `Tab`, such as `data-testid`. */
  tabProps?: Omit<TabProps, 'value' | 'label'> & {
    /** Test id for the tab. */
    'data-testid'?: string;
  };
}

/** Previous and next actions for the pane, usually the result of `useDetailsPaneNavigation`. */
export type DetailsPaneNavigation = Pick<
  UseDetailsPaneNavigationResult,
  'isFirst' | 'isLast' | 'next' | 'previous'
>;

export interface DetailsPaneLabels {
  /** Tooltip and accessible name of the button that opens the previous item. */
  previous: string;
  /** Tooltip and accessible name of the button that opens the next item. */
  next: string;
  /** Tooltip and accessible name of the button that closes the pane. */
  close: string;
}

/** Props for an icon button in the pane header; `data-testid` is allowed. */
export type DetailsPaneButtonProps = Omit<
  IconButtonProps,
  'onClick' | 'disabled' | 'children'
> & {
  /** Test id for the button. */
  'data-testid'?: string;
};

export interface DetailsPaneProps extends Omit<PaperProps, 'children'> {
  /** Whether the pane is up; it slides in from the bottom when this turns true. */
  open: boolean;
  /** Called when the user closes the pane with the close button or the Escape key. */
  onClose: () => void;
  /** Tabs in the order they appear; only the selected tab's content is rendered. */
  tabs: DetailsPaneTab[];
  /** Selected tab when the parent controls it; leave it out to let the pane keep its own. */
  tab?: string;
  /**
   * Tab the pane shows when it opens, when the parent does not control `tab`.
   * @default the first tab
   */
  defaultTab?: string;
  /** Called with the tab the user selects. */
  onTabChange?: (value: string) => void;
  /** Previous and next actions; the arrow buttons only show when this is set. */
  navigation?: DetailsPaneNavigation;
  /**
   * Tooltips and accessible names of the header buttons; name the item, such as "Next query".
   * @default { previous: 'Previous', next: 'Next', close: 'Close details' }
   */
  labels?: Partial<DetailsPaneLabels>;
  /** Props for the previous button, such as `data-testid`. */
  previousButtonProps?: DetailsPaneButtonProps;
  /** Props for the next button, such as `data-testid`. */
  nextButtonProps?: DetailsPaneButtonProps;
  /** Props for the close button, such as `data-testid`. */
  closeButtonProps?: DetailsPaneButtonProps;
  /** Props for the area under the tabs that scrolls the tab content. */
  contentProps?: BoxProps;
  /** Props for the MUI `Slide` that moves the pane in and out, such as `timeout`. */
  slideProps?: Omit<SlideProps, 'in' | 'children'>;
}
