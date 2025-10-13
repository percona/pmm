import { PMM_TITLE } from 'lib/constants';

export const updateDocumentTitle = (title?: string) => {
  if (title === PMM_TITLE) {
    return;
  }

  document.title = title ? `${title} - ${PMM_TITLE}` : PMM_TITLE;
};
