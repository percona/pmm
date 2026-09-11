import { downloadTextFile } from './file.utils';

describe('downloadTextFile', () => {
  const createObjectURLMock = vi.fn<(blob: Blob) => string>(
    () => 'blob:mock-url'
  );
  const revokeObjectURLMock = vi.fn();

  beforeEach(() => {
    createObjectURLMock.mockClear();
    revokeObjectURLMock.mockClear();
    URL.createObjectURL = createObjectURLMock;
    URL.revokeObjectURL = revokeObjectURLMock;
  });

  it('creates and clicks a download link with the given filename', () => {
    const clickMock = vi.fn();
    const anchor = document.createElement('a');
    anchor.click = clickMock;
    const createElementSpy = vi
      .spyOn(document, 'createElement')
      .mockReturnValue(anchor);

    downloadTextFile('my-template.yaml', 'name: my-template\n');

    expect(createObjectURLMock).toHaveBeenCalledTimes(1);
    const [blob] = createObjectURLMock.mock.calls[0];
    expect(blob.type).toBe('application/x-yaml');

    expect(createElementSpy).toHaveBeenCalledWith('a');
    expect(anchor.download).toBe('my-template.yaml');
    expect(anchor.href).toBe('blob:mock-url');
    expect(clickMock).toHaveBeenCalledTimes(1);
    expect(revokeObjectURLMock).toHaveBeenCalledWith('blob:mock-url');

    createElementSpy.mockRestore();
  });

  it('defaults the blob mime type to application/x-yaml', () => {
    downloadTextFile('a.yaml', 'content');
    const [blob] = createObjectURLMock.mock.calls[0];
    expect(blob.type).toBe('application/x-yaml');
  });

  it('accepts a custom mime type', () => {
    downloadTextFile('a.txt', 'content', 'text/plain');
    const [blob] = createObjectURLMock.mock.calls[0];
    expect(blob.type).toBe('text/plain');
  });
});
