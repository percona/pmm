import { parseQanDetailsTab } from './qanSectionTabs';

describe('parseQanDetailsTab', () => {
  it('maps legacy explain and plan tabs to explainPlan', () => {
    expect(parseQanDetailsTab('explain')).toBe('explainPlan');
    expect(parseQanDetailsTab('plan')).toBe('explainPlan');
  });

  it('defaults unknown tabs to details', () => {
    expect(parseQanDetailsTab('invalid')).toBe('details');
    expect(parseQanDetailsTab(null)).toBe('details');
  });

  it('accepts current section tab ids', () => {
    expect(parseQanDetailsTab('aiInsights')).toBe('aiInsights');
    expect(parseQanDetailsTab('explainPlan')).toBe('explainPlan');
  });
});
