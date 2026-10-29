import React from 'react';
import { ToolbarButton } from '@grafana/ui';
import { ThemeProvider } from 'contexts/theme';
import { openSearch } from 'compat/search';

const ToolbarSearchButton = () => (
  <ThemeProvider>
    <ToolbarButton
      aria-label="Search"
      iconOnly
      icon="search"
      onClick={() => openSearch()}
      data-testid="pmm-toolbar-search-button"
    />
  </ThemeProvider>
);

export default ToolbarSearchButton;
