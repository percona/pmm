import { SEARCH_DIALOG_TEST_ID } from './search.constants';

// Popovers such as NavFlyout are not modal, so they never block search.
export const isOtherModalOpen = () =>
  Array.from(
    document.querySelectorAll('[role="dialog"][aria-modal="true"]')
  ).some(
    (element) => !element.closest(`[data-testid="${SEARCH_DIALOG_TEST_ID}"]`)
  );
