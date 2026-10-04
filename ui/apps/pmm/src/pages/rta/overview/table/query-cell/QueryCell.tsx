import Box from '@mui/material/Box';
import { CodeBlock } from '@percona/peak-ui';
import { FC, useLayoutEffect, useRef, useState } from 'react';
import { CodeLanguage } from 'types/util.types';
import { codeBlockLanguage } from '../OverviewTable.utils';

export interface Props {
  query: string;
  language?: CodeLanguage;
}

// Fade the clipped text out toward the right edge as a cut-off affordance.
// Masking the content (not overlaying a gradient) keeps it independent of the
// prism scheme's background color in either mode.
const fadeMask =
  'linear-gradient(to right, black calc(100% - 48px), transparent)';

const fadeSx = {
  '& > code': {
    display: 'block',
    maskImage: fadeMask,
    maskRepeat: 'no-repeat',
    maskSize: '100% 100%',
    WebkitMaskImage: fadeMask,
    WebkitMaskRepeat: 'no-repeat',
    WebkitMaskSize: '100% 100%',
  },
};

// The fade is only applied while the text is actually clipped. Applied always, it ate the
// last characters of a statement that fits ("COMMI"), because a table column sized to its
// content leaves the text exactly as wide as the block the mask covers.
const useIsOverflowing = (content: string) => {
  const ref = useRef<HTMLPreElement>(null);
  const [overflowing, setOverflowing] = useState(false);

  useLayoutEffect(() => {
    const element = ref.current;
    if (!element) {
      return;
    }

    const update = () =>
      setOverflowing(element.scrollWidth > element.clientWidth);
    update();

    if (typeof ResizeObserver === 'undefined') {
      return;
    }

    const observer = new ResizeObserver(update);
    observer.observe(element);

    return () => observer.disconnect();
  }, [content]);

  return { ref, overflowing };
};

// Full-width container + hidden overflow so the block's frame and border
// always render inside the cell instead of getting cut off by it
const QueryCell: FC<Props> = ({ query, language = 'mongodb' }) => {
  const content = query
    .replace(/[\n\r\t]/g, '')
    .replace(/\s{2,}/g, ' ')
    .trim();
  const { ref, overflowing } = useIsOverflowing(content);

  return (
    <Box sx={{ width: '100%', minWidth: 0 }}>
      <CodeBlock
        ref={ref}
        content={content}
        language={codeBlockLanguage(language)}
        data-overflowing={overflowing}
        sx={{
          overflow: 'hidden',
          ...(overflowing ? fadeSx : {}),
        }}
      />
    </Box>
  );
};

export default QueryCell;
