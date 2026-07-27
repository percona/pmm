import { render, act } from '@testing-library/react';
import { FC, useState } from 'react';
import { describe, it, expect, vi, afterEach } from 'vitest';
import messenger from 'lib/messenger';
import { useMessengerListener } from './useMessengerListener';

const dispatch = (type: string) =>
  act(() => {
    window.dispatchEvent(new MessageEvent('message', { data: { type } }));
  });

afterEach(() => {
  vi.restoreAllMocks();
});

const Subscriber: FC<{ onSettingsChanged: () => void; enabled?: boolean }> = ({
  onSettingsChanged,
  enabled,
}) => {
  useMessengerListener('SETTINGS_CHANGED', onSettingsChanged, { enabled });
  return null;
};

describe('useMessengerListener', () => {
  it('subscribes while mounted and unsubscribes on unmount', () => {
    const onMessage = vi.fn();
    const { unmount } = render(<Subscriber onSettingsChanged={onMessage} />);

    dispatch('SETTINGS_CHANGED');
    expect(onMessage).toHaveBeenCalledTimes(1);

    unmount();
    dispatch('SETTINGS_CHANGED');
    expect(onMessage).toHaveBeenCalledTimes(1);
  });

  it('calls the latest handler without resubscribing', () => {
    const addListener = vi.spyOn(messenger, 'addListener');
    const seen: number[] = [];

    const Counter: FC = () => {
      const [count, setCount] = useState(0);
      useMessengerListener('SETTINGS_CHANGED', () => seen.push(count));
      return <button onClick={() => setCount((c) => c + 1)}>{count}</button>;
    };

    const { getByRole } = render(<Counter />);

    dispatch('SETTINGS_CHANGED');
    act(() => getByRole('button').click());
    act(() => getByRole('button').click());
    dispatch('SETTINGS_CHANGED');

    expect(seen).toEqual([0, 2]);
    expect(addListener).toHaveBeenCalledTimes(1);
  });

  it('does not subscribe while disabled', () => {
    const onMessage = vi.fn();
    const { rerender } = render(
      <Subscriber onSettingsChanged={onMessage} enabled={false} />
    );

    dispatch('SETTINGS_CHANGED');
    expect(onMessage).not.toHaveBeenCalled();

    rerender(<Subscriber onSettingsChanged={onMessage} enabled />);
    dispatch('SETTINGS_CHANGED');
    expect(onMessage).toHaveBeenCalledTimes(1);
  });
});
