import { createTheme } from '@grafana/data';
import { LOCATORS } from 'lib/constants';
import { applyInvertedTooltipStyles, getInvertedTooltipCss } from './tooltips';

const dark = createTheme({ colors: { mode: 'dark' } });
const light = createTheme({ colors: { mode: 'light' } });

describe('getInvertedTooltipCss', () => {
  it('gives tooltips the light theme colours on a dark page', () => {
    const css = getInvertedTooltipCss(dark);
    const { background, text } = light.components.tooltip;

    expect(css).toContain(`${LOCATORS.tooltip} {`);
    expect(css).toContain(`background-color: ${background};`);
    expect(css).toContain(`border-color: ${background};`);
    expect(css).toContain(`color: ${text};`);
    expect(css).toContain(
      `div:has(> ${LOCATORS.tooltip}) > svg {\n      fill: ${background};`
    );
    expect(css).toContain(`color: ${light.colors.text.link};`);
    // Grafana's global em rule sets the page theme's text colour
    expect(css).toContain(`${LOCATORS.tooltip} em {`);
    expect(css).not.toContain(dark.components.tooltip.background);
  });

  it('gives tooltips the dark theme colours on a light page', () => {
    const css = getInvertedTooltipCss(light);
    const { background, text } = dark.components.tooltip;

    expect(css).toContain(`background-color: ${background};`);
    expect(css).toContain(`color: ${text};`);
    expect(css).toContain(`fill: ${background};`);
    expect(css).toContain(`color: ${dark.colors.text.link};`);
  });
});

describe('applyInvertedTooltipStyles', () => {
  afterEach(() => {
    document.head.innerHTML = '';
    document.body.innerHTML = '';
  });

  it('keeps one style tag and rewrites it on theme change', () => {
    applyInvertedTooltipStyles(dark);
    applyInvertedTooltipStyles(light);

    const styles = document.head.querySelectorAll(
      'style#pmm-inverted-tooltips'
    );

    expect(styles).toHaveLength(1);
    expect(styles[0].textContent).toBe(getInvertedTooltipCss(light));
  });

  it('styles a Grafana tooltip', () => {
    applyInvertedTooltipStyles(dark);
    const tooltip = document.createElement('div');
    tooltip.setAttribute('role', 'tooltip');
    tooltip.dataset.testid = 'data-testid tooltip';
    document.body.appendChild(tooltip);

    expect(getComputedStyle(tooltip).backgroundColor).toBe(
      'rgb(255, 255, 255)'
    );
  });

  it('keeps markdown links readable inside the tooltip', () => {
    // Grafana's global element styles colour markdown links with the page theme
    const globals = document.createElement('style');
    globals.textContent = `.markdown-html a { color: ${dark.colors.text.link}; }`;
    document.head.appendChild(globals);
    applyInvertedTooltipStyles(dark);

    const tooltip = document.createElement('div');
    tooltip.setAttribute('role', 'tooltip');
    tooltip.dataset.testid = 'data-testid tooltip';
    const markdown = document.createElement('div');
    markdown.className = 'markdown-html';
    const link = document.createElement('a');
    markdown.appendChild(link);
    tooltip.appendChild(markdown);
    document.body.appendChild(tooltip);

    expect(getComputedStyle(link).color).toBe('rgb(31, 98, 224)');
  });
});
