import React, { useEffect, useRef } from 'react';
import highlight from 'highlight.js';
import { useStyles2 } from '@grafana/ui';
import { cx } from '@emotion/css';
import { HighlightProps } from './Highlight.types';
import { getStyles } from './Highlight.styles';

export const Highlight = ({
  children, className, language, mode,
}: HighlightProps) => {
  const styles = useStyles2(getStyles, mode);
  const ref = useRef<HTMLElement>(null);

  useEffect(() => {
    if (ref.current != null) {
      highlight.highlightElement(ref.current);
    }
  }, [ref]);

  return (
    <pre
      data-testid="highlight-pre"
      className={cx(className, styles.highlightWrapper)}
    >
      <code
        data-testid="highlight-code"
        className={`language-${language}`}
        ref={ref}
      >
        {children}
      </code>
    </pre>
  );
};
