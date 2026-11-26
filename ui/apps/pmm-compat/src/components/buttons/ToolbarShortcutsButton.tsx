import React from 'react';
import { ToolbarButton } from '@grafana/ui';
import { ThemeProvider } from 'contexts/theme';
import { triggerShortcut } from 'lib/utils';

const ToolbarShortcutsButton = () => (
  <ThemeProvider>
    <ToolbarButton
      iconOnly
      icon="keyboard"
      onClick={() => triggerShortcut('view-shortcuts')}
      data-testid="pmm-toolbar-shortcuts-button"
      tooltip="View shortcuts"
    />
  </ThemeProvider>
);

export default ToolbarShortcutsButton;
