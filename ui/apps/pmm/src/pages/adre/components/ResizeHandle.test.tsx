import { fireEvent, render, screen } from '@testing-library/react';
import { ResizeHandle } from './ResizeHandle';

describe('ResizeHandle', () => {
  it('follows the mouse within its bounds', () => {
    const onResize = vi.fn();
    render(
      <ResizeHandle
        side="left"
        width={260}
        min={200}
        max={440}
        onResize={onResize}
        label="Resize chats"
      />
    );
    const handle = screen.getByRole('separator', { name: 'Resize chats' });
    fireEvent.mouseDown(handle, { clientX: 300 });
    fireEvent.mouseMove(document, { clientX: 340 });
    expect(onResize).toHaveBeenLastCalledWith(300);
    fireEvent.mouseMove(document, { clientX: 900 });
    expect(onResize).toHaveBeenLastCalledWith(440);
    fireEvent.mouseUp(document);
    fireEvent.mouseMove(document, { clientX: 100 });
    expect(onResize).toHaveBeenCalledTimes(2);
  });

  it('grows a panel on its right when dragged left', () => {
    const onResize = vi.fn();
    render(
      <ResizeHandle
        side="right"
        width={340}
        min={280}
        max={560}
        onResize={onResize}
        label="Resize side panel"
      />
    );
    const handle = screen.getByRole('separator', { name: 'Resize side panel' });
    fireEvent.mouseDown(handle, { clientX: 1000 });
    fireEvent.mouseMove(document, { clientX: 960 });
    expect(onResize).toHaveBeenLastCalledWith(380);
    fireEvent.mouseUp(document);
  });

  it('moves with the arrow keys and jumps to its bounds', () => {
    const onResize = vi.fn();
    render(
      <ResizeHandle
        side="left"
        width={260}
        min={200}
        max={440}
        onResize={onResize}
        label="Resize chats"
      />
    );
    const handle = screen.getByRole('separator', { name: 'Resize chats' });
    expect(handle).toHaveAttribute('aria-valuenow', '260');
    fireEvent.keyDown(handle, { key: 'ArrowRight' });
    expect(onResize).toHaveBeenLastCalledWith(276);
    fireEvent.keyDown(handle, { key: 'ArrowLeft' });
    expect(onResize).toHaveBeenLastCalledWith(244);
    fireEvent.keyDown(handle, { key: 'End' });
    expect(onResize).toHaveBeenLastCalledWith(440);
    fireEvent.keyDown(handle, { key: 'Home' });
    expect(onResize).toHaveBeenLastCalledWith(200);
  });
});
