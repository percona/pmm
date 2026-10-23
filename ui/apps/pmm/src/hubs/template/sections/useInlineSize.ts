import { useLayoutEffect, useRef, useState } from 'react';

/** An element's content width, kept current; undefined until measured, and where nothing can measure it. */
export const useInlineSize = <T extends HTMLElement>() => {
  const ref = useRef<T>(null);
  const [width, setWidth] = useState<number>();

  useLayoutEffect(() => {
    const element = ref.current;

    if (!element || typeof ResizeObserver === 'undefined') {
      return undefined;
    }
    const observer = new ResizeObserver(([entry]) =>
      setWidth(entry.contentRect.width || undefined)
    );

    observer.observe(element);

    return () => observer.disconnect();
  }, []);

  return [ref, width] as const;
};
