import React from 'react';
import { createRoot } from 'react-dom/client';
import { ToolbarSearchButton, ToolbarShortcutsButton } from 'components/buttons';
import { LOCATORS } from 'lib/constants';
import { waitForElement } from 'lib/utils';

export const adjustToolbar = async () => {
  const toolbar = await waitForElement(LOCATORS.toolbar, 5000);

  if (!toolbar) {
    return;
  }

  // Appends toolbar button to trigger search and keyboard shortcuts modal
  const pre = document.createElement('div');
  const post = document.createElement('div');

  toolbar.prepend(pre);
  toolbar.append(post);

  createRoot(pre).render(<ToolbarSearchButton />);

  createRoot(post).render(<ToolbarShortcutsButton />);
};
