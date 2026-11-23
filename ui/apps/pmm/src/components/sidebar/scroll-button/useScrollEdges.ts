import { useEffect, useState } from 'react';

/** Tracks whether a container can scroll up or down, from markers at its content's ends. */
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
