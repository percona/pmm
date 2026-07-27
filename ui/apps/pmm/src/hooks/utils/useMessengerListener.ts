import { Message, MessageType } from '@pmm/shared';
import messenger from 'lib/messenger';
import { useEffect, useRef } from 'react';

/**
 * Subscribe to a cross-frame message for as long as the component is mounted.
 *
 * The handler is mirrored into a ref on every render, so it always sees current
 * props and state without the subscription having to churn — the effect depends
 * only on the message type. Unsubscribing removes just this listener, never
 * anyone else's.
 */
export const useMessengerListener = <T extends MessageType, V = unknown>(
  type: T,
  handler: (message: Message<T, V>) => void,
  options?: { enabled?: boolean }
) => {
  const handlerRef = useRef(handler);
  handlerRef.current = handler;

  const enabled = options?.enabled ?? true;

  useEffect(() => {
    if (!enabled) {
      return;
    }

    return messenger.addListener<T, V>({
      type,
      onMessage: (message) => handlerRef.current(message),
    });
  }, [type, enabled]);
};
