import React from 'react';
import { ToolbarButton } from '@grafana/ui';
import { ThemeProvider } from 'contexts/theme';
import { triggerShortcut } from 'lib/utils';

const ToolbarSearchButton = () => (
  <ThemeProvider>
    <ToolbarButton
      iconOnly
      icon="search"
      onClick={() => triggerShortcut('search')}
      data-testid="pmm-toolbar-search-button"
    />
  </ThemeProvider>
);

export default ToolbarSearchButton;
