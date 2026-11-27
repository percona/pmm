import { FC } from 'react';
import { useQanDatabaseType } from '../../hooks/useQanDatabaseType';
import { useQanPanelState } from '../../hooks/useQanPanelState';
import { QanExplainTab } from './QanExplainTab';
import { QanPlanTab } from './QanPlanTab';

/** Figma “Explain Plan” — MySQL EXPLAIN or PostgreSQL plan in one section tab. */
export const QanExplainPlanTab: FC = () => {
  const state = useQanPanelState();
  const databaseType = useQanDatabaseType(state.labels, state.database);

  if (databaseType === 'postgresql') {
    return <QanPlanTab />;
  }

  return <QanExplainTab />;
};
