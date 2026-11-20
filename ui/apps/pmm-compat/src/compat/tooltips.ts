import { createTheme, type GrafanaTheme2 } from '@grafana/data';
import { LOCATORS } from 'lib/constants';

const STYLE_ID = 'pmm-inverted-tooltips';

/** Gives Grafana's Tooltip the opposite theme's tooltip colours, like Peak UI's Tooltip in the PMM UI. */
export const getInvertedTooltipCss = (theme: GrafanaTheme2): string => {
  const tip = LOCATORS.tooltip;
  // Not the components.tooltip theme token: Toggletips and graph hover boxes read it too.
  const inverted = createTheme({
    colors: { mode: theme.isDark ? 'light' : 'dark' },
  });
  const { background, text } = inverted.components.tooltip;

  return `
    ${tip} {
      background-color: ${background};
      border-color: ${background};
      color: ${text};
    }

    div:has(> ${tip}) > svg {
      fill: ${background};
    }

    ${tip} a,
    ${tip} em {
      color: inherit;
    }

    ${tip} .markdown-html a,
    ${tip} a.external-link {
      color: ${inverted.colors.text.link};
    }

    ${tip} code {
      background-color: ${inverted.colors.background.primary};
      border-color: ${inverted.colors.border.medium};
      color: ${inverted.colors.text.primary};
    }
  `;
};

export const applyInvertedTooltipStyles = (theme: GrafanaTheme2) => {
  let style = document.getElementById(STYLE_ID);

  if (!style) {
    style = document.createElement('style');
    style.id = STYLE_ID;
    document.head.appendChild(style);
  }

  style.textContent = getInvertedTooltipCss(theme);
};
