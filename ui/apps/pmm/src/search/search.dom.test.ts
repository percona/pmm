import { SEARCH_DIALOG_TEST_ID } from './search.constants';
import { isOtherModalOpen } from './search.dom';

const addDialog = ({
  modal = true,
  own = false,
}: { modal?: boolean; own?: boolean } = {}) => {
  const dialog = document.createElement('div');
  dialog.setAttribute('role', 'dialog');

  if (modal) {
    dialog.setAttribute('aria-modal', 'true');
  }

  if (own) {
    const root = document.createElement('div');
    root.dataset.testid = SEARCH_DIALOG_TEST_ID;
    root.append(dialog);
    document.body.append(root);
  } else {
    document.body.append(dialog);
  }
};

describe('isOtherModalOpen', () => {
  afterEach(() => {
    document.body.innerHTML = '';
  });

  it('ignores search’s own dialog', () => {
    addDialog({ own: true });

    expect(isOtherModalOpen()).toBe(false);
  });

  it('sees another modal dialog, also next to search’s own', () => {
    addDialog({ own: true });
    addDialog();

    expect(isOtherModalOpen()).toBe(true);
  });

  it('ignores a dialog that is not modal, such as a popover', () => {
    addDialog({ modal: false });

    expect(isOtherModalOpen()).toBe(false);
  });
});
