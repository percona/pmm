import React, { Fragment } from 'react';
import { Highlight as PrismHighlight, themes } from 'prism-react-renderer';
import { useStyles2, useTheme2 } from '@grafana/ui';
import { cx } from '@emotion/css';
import { HighlightProps } from './Highlight.types';
import { getStyles } from './Highlight.styles';

export const Highlight = ({ children, className, language }: HighlightProps) => {
  const styles = useStyles2(getStyles);
  // Same token themes as Peak UI's CodeBlock
  const prismTheme = useTheme2().isDark ? themes.okaidia : themes.nightOwlLight;

  return (
    <PrismHighlight code={children} language={language} theme={prismTheme}>
      {({ tokens, getLineProps, getTokenProps }) => (
        <pre data-testid="highlight-pre" className={cx(className, styles.highlightWrapper)}>
          <code data-testid="highlight-code" className={`language-${language}`}>
            {/* Lines are joined with real newlines so the text copies exactly as given */}
            {tokens.map((line, lineIndex) => (
              // eslint-disable-next-line react/no-array-index-key
              <Fragment key={lineIndex}>
                {lineIndex > 0 ? '\n' : null}
                <span {...getLineProps({ line })}>
                  {line.map((token, tokenIndex) => (
                    // eslint-disable-next-line react/no-array-index-key
                    <span key={tokenIndex} {...getTokenProps({ token: token.empty ? { ...token, content: '' } : token })} />
                  ))}
                </span>
              </Fragment>
            ))}
          </code>
        </pre>
      )}
    </PrismHighlight>
  );
};
