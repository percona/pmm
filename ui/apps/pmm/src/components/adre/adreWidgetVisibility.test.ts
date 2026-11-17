import { showsAdreWidget } from './adreWidgetVisibility';

describe('showsAdreWidget', () => {
  it('hides the floating chat on pages with their own chat', () => {
    expect(showsAdreWidget('/assistant')).toBe(false);
    expect(showsAdreWidget('/qan')).toBe(false);
  });

  it('shows it elsewhere', () => {
    expect(showsAdreWidget('/graph/d/pmm-home')).toBe(true);
    expect(showsAdreWidget('/configuration/assistant')).toBe(true);
    expect(showsAdreWidget('/assistants-guide')).toBe(true);
    expect(showsAdreWidget('/qan/ai-insights')).toBe(true);
  });
});
