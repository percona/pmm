import { useEffect, useState } from 'react';

/**
 * Tracks whether a scroll container can scroll further up or down, using
 * markers placed at the very start and end of its content. The browser reports
 * a marker entering or leaving view, so nothing runs while scrolling. The refs
 * are callback refs, so a container or marker that remounts is observed afresh.
 */
export const useScrollEdges = <
  Root extends HTMLElement,
  Marker extends HTMLElement,
>() => {
  const [root, setRoot] = useState<Root | null>(null);
  const [start, setStart] = useState<Marker | null>(null);
  const [end, setEnd] = useState<Marker | null>(null);
  const [canScrollUp, setCanScrollUp] = useState(false);
  const [canScrollDown, setCanScrollDown] = useState(false);

  useEffect(() => {
    if (!root || !start || !end) {
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach(({ target, isIntersecting }) => {
          // A marker that was just removed reports leaving view; its
          // replacement reports for itself.
          if (!target.isConnected) {
            return;
          }

          if (target === start) {
            setCanScrollUp(!isIntersecting);
          } else {
            setCanScrollDown(!isIntersecting);
          }
        });
      },
      { root }
    );

    observer.observe(start);
    observer.observe(end);

    return () => observer.disconnect();
  }, [root, start, end]);

  return {
    root,
    rootRef: setRoot,
    startRef: setStart,
    endRef: setEnd,
    canScrollUp,
    canScrollDown,
  };
};
