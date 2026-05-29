import type { QanDetailsTab } from 'types/qan.types';

/** Section tab order and Figma labels (Query Fingerprint panel). */
export const QAN_SECTION_TAB_ORDER: QanDetailsTab[] = [
  'details',
  'examples',
  'explainPlan',
  'tables',
  'aiInsights',
];

export const QAN_SECTION_TAB_LABELS: Record<QanDetailsTab, string> = {
  details: 'Details',
  examples: 'Examples',
  explainPlan: 'Explain Plan',
  tables: 'Tables',
  aiInsights: 'Get AI Insights',
};

export function parseQanDetailsTab(raw: string | null): QanDetailsTab {
  if (raw === 'explain' || raw === 'plan') return 'explainPlan';
  const allowed: QanDetailsTab[] = [
    'details',
    'examples',
    'explainPlan',
    'tables',
    'aiInsights',
  ];
  return (allowed.includes(raw as QanDetailsTab) ? raw : 'details') as QanDetailsTab;
}
