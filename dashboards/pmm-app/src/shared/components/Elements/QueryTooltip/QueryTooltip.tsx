import { Tooltip } from 'antd';
import React, { useMemo } from 'react';
import { createTheme, ThemeContext } from '@grafana/data';
import { useStyles2, useTheme2 } from '@grafana/ui';
import sqlFormatter from 'sql-formatter';
import { Highlight } from 'shared/components/Hightlight/Highlight';
import { getStyles } from './QueryTooltip.styles';

interface QueryTooltipProps {
  query: string;
  queryId?: string;
  show?: boolean;
  children: any;
  float?: string;
}
export const QueryTooltip = ({
  query, queryId, children, show = true, float = 'none',
}: QueryTooltipProps) => {
  // Tooltips render in the opposite theme to the page, like Peak UI's Tooltip in the PMM UI
  const { isDark } = useTheme2();
  const tooltipTheme = useMemo(() => createTheme({ colors: { mode: isDark ? 'light' : 'dark' } }), [isDark]);
  const styles = useStyles2(getStyles, tooltipTheme);

  return show ? (
    <Tooltip
      placement="left"
      mouseEnterDelay={0}
      data-testid="query-tooltip"
      overlayClassName={styles.queryTooltip(float)}
      title={() => (
        <ThemeContext.Provider value={tooltipTheme}>
          <div className={styles.queryTooltipDataWrapper}>
            {queryId ? <h5 className={styles.queryId}>{`Query Id: ${queryId}`}</h5> : null}
            {query ? <Highlight language="sql">{sqlFormatter.format(query, { language: 'pl/sql' })}</Highlight> : null}
          </div>
        </ThemeContext.Provider>
      )}
    >
      {children}
    </Tooltip>
  ) : (
    <>{children}</>
  );
};
