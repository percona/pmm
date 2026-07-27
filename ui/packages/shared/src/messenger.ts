import { Message, MessageListener, MessageType } from './types';
import { v4 as uuidv4 } from 'uuid';

export type Unsubscribe = () => void;

/** Messages a peer sends once it has wired up its own listeners. */
export const READY_MESSAGE_TYPES: ReadonlySet<MessageType> =
  new Set<MessageType>(['MESSENGER_READY', 'GRAFANA_READY']);

/**
 * State-sync messages that are still meaningful when they arrive late, so they
 * are buffered while no peer is reachable. Everything else is dropped: replaying
 * a handshake is meaningless, and a request/response needs a live peer.
 */
export const QUEUEABLE_MESSAGE_TYPES: ReadonlySet<MessageType> =
  new Set<MessageType>([
    'CHANGE_THEME',
    'LOCATION_CHANGE',
    'SETTINGS_CHANGED',
    'FRONTEND_SETTINGS_CHANGED',
    'SERVICE_ADDED',
    'SERVICE_DELETED',
    'TIMEZONE_CHANGED',
  ]);

/**
 * The outbox is keyed by message type, so it can never outgrow the queueable
 * set. This cap only guards senders that opt in explicitly via `{ queue: true }`.
 */
const MAX_OUTBOX = 32;

/** Same-origin, the default `postMessage` uses when given no target origin. */
const SAME_ORIGIN = '/';

export class CrossFrameMessenger {
  private source?: string;
  private hostWindow?: Window =
    typeof window === 'undefined' ? undefined : window;
  private targetOrigin?: string;
  private targetWindow?: Window;
  private targetResolver?: () => Window | null | undefined;
  private fallbackSelector?: string;
  private lastResolvedTarget?: Window;
  private targetReady = false;
  private registered = false;
  private eventListener = (e: MessageEvent) => this.onMessageReceived(e);

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private listeners = new Map<MessageType, Set<MessageListener<any, any>>>();
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private outbox = new Map<MessageType, Message<any, any>>();

  constructor(source?: string) {
    this.source = source;
  }

  setTargetOrigin(origin: string) {
    this.targetOrigin = origin;
    return this;
  }

  setWindow(window: Window) {
    if (window === this.hostWindow) {
      return this;
    }

    const wasRegistered = this.registered;
    this.detach();
    this.hostWindow = window;

    if (wasRegistered) {
      this.register();
    }

    return this;
  }

  setTargetWindow(window: Window, fallbackSelector?: string) {
    this.targetWindow = window;
    this.fallbackSelector = fallbackSelector;
    return this;
  }

  /**
   * Resolve the target lazily on every send. Prefer this over `setTargetWindow`
   * whenever the target frame can be unmounted and remounted: a cached
   * `contentWindow` goes stale the moment its iframe leaves the DOM.
   */
  setTargetResolver(resolver: () => Window | null | undefined) {
    this.targetResolver = resolver;
    return this;
  }

  /** Idempotent — safe to call from module init and again from a component. */
  register() {
    if (this.registered || !this.hostWindow) {
      return this;
    }

    this.hostWindow.addEventListener('message', this.eventListener);
    this.registered = true;
    return this;
  }

  /**
   * @deprecated Tearing the messenger down destroys listeners owned by other
   * callers. Unsubscribe your own listener with the callback `addListener`
   * returns; use {@link destroy} only when the whole channel is going away.
   */
  unregister() {
    this.destroy();
  }

  /** Detach from the host window and forget all listeners and queued messages. */
  destroy() {
    this.detach();
    this.clearListeners();
    this.outbox.clear();
    this.targetReady = false;
  }

  clearListeners() {
    this.listeners.clear();
  }

  addListener<T extends MessageType, V>(
    listener: MessageListener<T, V>
  ): Unsubscribe {
    const listeners = this.listeners.get(listener.type) ?? new Set();
    listeners.add(listener);
    this.listeners.set(listener.type, listeners);

    return () => this.removeListener(listener);
  }

  removeListener<T extends MessageType, V>(listener: MessageListener<T, V>) {
    const listeners = this.listeners.get(listener.type);

    if (!listeners) {
      return;
    }

    listeners.delete(listener);

    if (!listeners.size) {
      this.listeners.delete(listener.type);
    }
  }

  onMessageReceived<T extends MessageType, V>(e: MessageEvent) {
    const message = e.data as Message<T, V>;

    if (!message?.type) {
      return;
    }

    // Dispatch over a snapshot: handlers may subscribe or unsubscribe while we
    // iterate (`waitForMessage` and `sendMessageWithResult` remove themselves).
    const listeners = this.listeners.get(message.type);
    if (listeners) {
      [...listeners].forEach((listener) => listener.onMessage(message));
    }

    // Only a handshake drains the outbox. Other tooling (Vite HMR, React
    // DevTools) posts into this window too, and flushing at them would send the
    // queue to a peer that is not listening.
    if (READY_MESSAGE_TYPES.has(message.type)) {
      // Resolve first: resolution re-arms `targetReady` when the frame changed.
      this.getWindow();
      this.targetReady = true;
      this.flushOutbox();
    }
  }

  /**
   * @returns whether the message was posted right away. `false` means it was
   * either queued for the next handshake or dropped.
   */
  sendMessage<T extends MessageType, V>(
    message: Message<T, V>,
    options?: { queue?: boolean }
  ): boolean {
    // provide source and id if not present
    const msg = {
      ...message,
      source: this.source,
      id: message.id || uuidv4(),
    };

    const target = this.getWindow();
    const queueable =
      options?.queue ?? QUEUEABLE_MESSAGE_TYPES.has(message.type);

    if (target) {
      target.postMessage(msg, this.targetOrigin ?? SAME_ORIGIN);

      // The frame exists, but the peer may still be booting and not listening
      // yet — keep the message so the handshake replays it.
      if (!this.targetReady && queueable) {
        this.enqueue(msg);
      }

      return true;
    }

    if (queueable) {
      this.enqueue(msg);
    }

    return false;
  }

  sendMessageWithResult = <U, T extends MessageType, V>(
    message: Message<T, V>,
    timeoutMs = 10_000
  ): Promise<U> =>
    new Promise((resolve, reject) => {
      const id = uuidv4();

      const timeoutId = setTimeout(() => {
        unsubscribe();
        reject(new Error(`Timed out waiting for a "${message.type}" reply`));
      }, timeoutMs);

      const unsubscribe = this.addListener<T, U>({
        type: message.type,
        onMessage: (received) => {
          if (received.id !== id) {
            return;
          }

          clearTimeout(timeoutId);
          unsubscribe();
          resolve(received.payload!);
        },
      });

      // A queued request would be answered against a stale caller, so fail fast
      // instead of leaving the caller hanging until the timeout.
      if (!this.sendMessage({ ...message, id }, { queue: false })) {
        clearTimeout(timeoutId);
        unsubscribe();
        reject(new Error(`No target frame to handle "${message.type}"`));
      }
    });

  waitForMessage = <T extends MessageType>(
    type: T,
    timeoutMs = 10_000
  ): Promise<void> =>
    new Promise((resolve, reject) => {
      const timeoutId = setTimeout(() => {
        unsubscribe();
        reject(new Error(`Timed out waiting for "${type}"`));
      }, timeoutMs);

      const unsubscribe = this.addListener<T, void>({
        type,
        onMessage: () => {
          clearTimeout(timeoutId);
          unsubscribe();
          resolve();
        },
      });
    });

  /** Whether a live target frame can be reached right now. */
  hasTarget() {
    return !!this.getWindow();
  }

  /** Whether the target frame has announced that its listeners are up. */
  isTargetReady() {
    return this.targetReady;
  }

  /**
   * Forget the cached target and require a fresh handshake. Call this when the
   * frame navigates in place, which keeps its `Window` identity but restarts
   * the peer.
   */
  invalidateTarget() {
    this.targetWindow = undefined;
    this.lastResolvedTarget = undefined;
    this.targetReady = false;
    return this;
  }

  flushOutbox() {
    if (!this.outbox.size) {
      return;
    }

    const target = this.getWindow();

    if (!target) {
      return;
    }

    const pending = [...this.outbox.values()];
    this.outbox.clear();
    pending.forEach((msg) =>
      target.postMessage(msg, this.targetOrigin ?? SAME_ORIGIN)
    );
  }

  private detach() {
    this.hostWindow?.removeEventListener('message', this.eventListener);
    this.registered = false;
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private enqueue(message: Message<any, any>) {
    // Re-inserting moves the entry to the back, so the flush order matches the
    // send order and only the newest message of each type survives.
    this.outbox.delete(message.type);
    this.outbox.set(message.type, message);

    while (this.outbox.size > MAX_OUTBOX) {
      const oldest = this.outbox.keys().next().value as MessageType;
      this.outbox.delete(oldest);
    }
  }

  private isUsable(win?: Window | null): win is Window {
    try {
      return !!win && !win.closed;
    } catch {
      // Cross-origin frames can throw on property access.
      return false;
    }
  }

  private getWindow(): Window | undefined {
    let target = this.isUsable(this.targetWindow)
      ? this.targetWindow
      : undefined;

    if (!target) {
      // Drop the stale handle so we never post into a destroyed frame.
      this.targetWindow = undefined;
      const resolved = this.targetResolver?.() ?? this.queryFallback();
      target = this.isUsable(resolved) ? resolved : undefined;
    }

    if (target !== this.lastResolvedTarget) {
      this.lastResolvedTarget = target;
      // A different frame has to announce itself before we trust it again.
      this.targetReady = false;
    }

    return target;
  }

  private queryFallback() {
    if (!this.fallbackSelector || typeof document === 'undefined') {
      return undefined;
    }

    return document.querySelector<HTMLIFrameElement>(this.fallbackSelector)
      ?.contentWindow;
  }
}
