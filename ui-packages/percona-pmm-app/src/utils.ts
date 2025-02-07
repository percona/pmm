import { DataLinkBuiltInVars, locationUtil, textUtil, urlUtil } from '@grafana/data';
import { config, getTemplateSrv } from '@grafana/runtime';
import { DashboardLink } from '@grafana/schema';

export const getLinkUrl = (link: Partial<DashboardLink>) => {
  let params: { [key: string]: any } = {};

  if (link.keepTime) {
    params[`\$${DataLinkBuiltInVars.keepTime}`] = true;
  }

  if (link.includeVars) {
    params[`\$${DataLinkBuiltInVars.includeVars}`] = true;
  }

  let url = locationUtil.assureBaseUrl(urlUtil.appendQueryToUrl(link.url || '', urlUtil.toUrlParams(params)));
  url = getTemplateSrv().replace(url);

  return config.disableSanitizeHtml ? url : textUtil.sanitizeUrl(url);
};

export const getLinkWithVariables = (url?: string): string => {
  if (url && isDashboardUrl(url) && isDashboardUrl(window.location.pathname)) {
    return getLinkUrl({
      url: url,
      keepTime: true,
      // Check if the DB type matches the current one used
      includeVars: checkDbType(url),
    });
  } else {
    return url ? url : '#';
  }
};

const isDashboardUrl = (url?: string) => url?.includes('/d/');

const checkDbType = (url: string): boolean => {
  const currentDB = window.location.pathname?.split('/')[3]?.split('-')[0];
  const urlDB = url?.split('/')[3]?.split('-')[0];

  return currentDB !== undefined && currentDB === urlDB;
};

export const appendCustomStyles = () => {
  const style = document.createElement('style');
  style.innerText = `
              #mega-menu-toggle,
          header > div:first-child,
          header > div:nth-child(2) > div:first-of-type,
          header div[class*=NavToolbar-actions] > div:last-of-type,
          button[title="Toggle top search bar"]
           {
            display: none;
          }`;
  document.head.appendChild(style);
};

export const isWithinIframe = () => window.self !== window.top;

export const log = (...msg: any[]) => console.log('pmm-app', ...msg);
