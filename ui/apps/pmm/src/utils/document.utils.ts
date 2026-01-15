import { PMM_TITLE } from 'lib/constants';

export const updateDocumentTitle = (title?: string) => {
  if (title === PMM_TITLE) {
    return;
  }

  if (!title) {
    document.title = PMM_TITLE;
  } else if (title.endsWith(PMM_TITLE)) {
    document.title = title;
  } else {
    document.title = `${title} - ${PMM_TITLE}`;
  }
};
