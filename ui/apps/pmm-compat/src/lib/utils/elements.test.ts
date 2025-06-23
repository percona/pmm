import { waitForElement } from './elements';

describe('utils/elements', () => {
  it('should return element when it exists', async () => {
    document.body.innerHTML = `
      <div class="existing-element"></div>`;

    const element = await waitForElement('.existing-element', 1000);

    expect(element).not.toBeNull();
    expect(element!.classList.contains('existing-element')).toBe(true);
  });

  it('should return null if element does not exist', async () => {
    const element = await waitForElement('.non-existent-element', 1000);

    expect(element).toBeNull();
  });
});
