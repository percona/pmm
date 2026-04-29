import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';
import { MarkdownBlock } from './MarkdownBlock';
import { SummaryBlock } from './SummaryBlock';
import { FindingBlock } from './FindingBlock';
import { QueryResultBlock } from './QueryResultBlock';
import { PanelBlock } from './PanelBlock';
import { LogsViewBlock } from './LogsViewBlock';
import { SlowQueryAnalysisBlock } from './SlowQueryAnalysisBlock';
import { TopQueriesBlock } from './TopQueriesBlock';
import { SchemaViewBlock } from './SchemaViewBlock';
import { RemediationStepsBlock } from './RemediationStepsBlock';
import { ImageBlock } from './ImageBlock';

export const BlockRenderer: FC<{ block: InvestigationBlock }> = ({ block }) => {
  switch (block.type) {
    case 'summary':
      return <SummaryBlock block={block} />;
    case 'markdown':
      return <MarkdownBlock block={block} />;
    case 'finding':
      return <FindingBlock block={block} />;
    case 'query_result':
      return <QueryResultBlock block={block} />;
    case 'single_panel':
    case 'panel_group':
    case 'logs_view':
      return block.type === 'logs_view' ? <LogsViewBlock block={block} /> : <PanelBlock block={block} />;
    case 'slow_query_analysis':
      return <SlowQueryAnalysisBlock block={block} />;
    case 'top_queries':
      return <TopQueriesBlock block={block} />;
    case 'schema_view':
      return <SchemaViewBlock block={block} />;
    case 'remediation_steps':
      return <RemediationStepsBlock block={block} />;
    case 'image':
      return <ImageBlock block={block} />;
    default:
      return (
        <MarkdownBlock
          block={{
            ...block,
            type: 'markdown',
            dataJson: block.dataJson ?? { content: `(Unsupported block type: ${block.type})` },
          }}
        />
      );
  }
};
