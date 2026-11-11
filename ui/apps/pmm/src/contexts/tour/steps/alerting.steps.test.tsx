import { render, screen } from '@testing-library/react';
import { ReactElement } from 'react';
import { StepType } from '@reactour/tour';
import {
  TEST_USER_ADMIN,
  TEST_USER_EDITOR,
  TEST_USER_VIEWER,
} from 'utils/testStubs';
import { addAlerting } from 'contexts/navigation/navigation.utils';
import { NAV_HOME_PAGE } from 'contexts/navigation/navigation.constants';
import { NavItem } from 'types/navigation.types';
import { User } from 'types/user.types';
import { getAlertingTourSteps } from './alerting.steps';
import { Messages } from './alerting.messages';

const selector = (id: string) => `[data-testid="navitem-${id}-list-item"]`;

const menuSelectors = (alerts: NavItem) =>
  (alerts.children || []).map((item) => selector(item.id));

const stepSelectors = (steps: StepType[]) => steps.map((step) => step.selector);

describe('getAlertingTourSteps', () => {
  const cases: [string, boolean, boolean, User][] = [
    ['admin', true, true, TEST_USER_ADMIN],
    ['editor', true, true, TEST_USER_EDITOR],
    ['viewer', true, true, TEST_USER_VIEWER],
    ['admin, alerting disabled', false, true, TEST_USER_ADMIN],
    ['admin, unified alerting disabled', true, false, TEST_USER_ADMIN],
    ['admin, both disabled', false, false, TEST_USER_ADMIN],
  ];

  it.each(cases)(
    'follows the Alerts menu for %s',
    (_, alertingEnabled, unifiedAlertingEnabled, user) => {
      const alerts = addAlerting(alertingEnabled, unifiedAlertingEnabled, user);

      const steps = getAlertingTourSteps([NAV_HOME_PAGE, alerts]);

      expect(stepSelectors(steps)).toEqual(menuSelectors(alerts));
    }
  );

  it('has a tip for every Alerts menu item', () => {
    const alerts = addAlerting(true, true, TEST_USER_ADMIN);

    for (const item of alerts.children || []) {
      expect(Messages[item.id]).toBeDefined();
    }
  });

  it('starts on Alert rules, which every user sees', () => {
    const steps = getAlertingTourSteps([
      addAlerting(false, false, TEST_USER_VIEWER),
    ]);

    expect(steps[0].selector).toBe(selector('alerts-rules'));
  });

  it('does not point at the removed Fired alerts page', () => {
    const steps = getAlertingTourSteps([
      addAlerting(true, true, TEST_USER_ADMIN),
    ]);

    expect(stepSelectors(steps)).not.toContain(selector('alerts-fired'));
  });

  it('uses the menu label as the tip title', () => {
    const alerts = addAlerting(true, true, TEST_USER_EDITOR);
    const steps = getAlertingTourSteps([alerts]);
    const templatesIndex = (alerts.children || []).findIndex(
      (item) => item.id === 'alerts-templates'
    );

    render(steps[templatesIndex].content as ReactElement);

    expect(screen.getByTestId('tour-step-title')).toHaveTextContent(
      'Templates'
    );
  });

  it('skips hidden menu items', () => {
    const alerts = addAlerting(true, true, TEST_USER_ADMIN);
    const withHidden = {
      ...alerts,
      children: (alerts.children || []).map((item) =>
        item.id === 'alerts-status' ? { ...item, hidden: true } : item
      ),
    };

    const steps = getAlertingTourSteps([withHidden]);

    expect(stepSelectors(steps)).not.toContain(selector('alerts-status'));
  });

  it('returns no steps without an Alerts menu', () => {
    expect(getAlertingTourSteps([NAV_HOME_PAGE])).toEqual([]);
  });
});
