import { DataLinkBuiltInVars, locationUtil, textUtil, urlUtil } from '@grafana/data';
import { config, getTemplateSrv } from '@grafana/runtime';
import { DashboardLink } from '@grafana/schema';

/**
 * Needs to be in sync with public/app/features/panel/panellinks/link_srv.ts LinkSrv.getLinkUrl
 */
const getLinkUrl = (link: Partial<DashboardLink>) => {
  let url = link.url ?? '';

  if (link.keepTime) {
    url = urlUtil.appendQueryToUrl(url, `\$${DataLinkBuiltInVars.keepTime}`);
  }

  if (link.includeVars) {
    url = urlUtil.appendQueryToUrl(url, `\$${DataLinkBuiltInVars.includeVars}`);
  }

  url = getTemplateSrv().replace(url);
  url = locationUtil.assureBaseUrl(url);

  return config.disableSanitizeHtml ? url : textUtil.sanitizeUrl(url);
};

export const getLinkWithVariables = (url?: string): string => {
  if (url && isDashboardUrl(url) && isDashboardUrl(window.location.pathname)) {
    const urlWithLinks = getLinkUrl({
      url: url,
      keepTime: true,
      // Check if the DB type matches the current one used
      includeVars: shouldIncludeVars(url),
      asDropdown: false,
      icon: '',
      tags: [],
      targetBlank: false,
      title: '',
      tooltip: '',
      type: 'link',
    });
    return cleanupVariables(urlWithLinks);
  } else {
    return url ? url : '#';
  }
};

const isDashboardUrl = (url?: string) => url?.includes('/d/');

export const shouldIncludeVars = (url: string): boolean => {
  const currentDB = getDbType(window.location.pathname);
  const targetDB = getDbType(url);

  if (currentDB === undefined || targetDB === undefined) {
    return false;
  }

  // enable variable sharing between same db types and db type -> os/node
  return currentDB === targetDB || targetDB === 'node';
};

const getDbType = (url: string): string | undefined => {
  const pathname = new URL(url, window.location.origin).pathname;
  // normalize to the dashboard uid
  const pathParts = pathname
    .replace('/pmm-ui', '')
    .replace('/next', '')
    .replace('/graph', '')
    .replace('/d/', '')
    .split('/');

  if (pathParts.length < 1 || !pathParts[0]) {
    return undefined;
  }

  const dashboardUid = pathParts[0];

  if (dashboardUid.includes('-')) {
    return dashboardUid.split('-')[0];
  }

  return dashboardUid;
};

export const cleanupVariables = (urlWithLinks: string) => {
  const [base, params] = urlWithLinks.split('?');

  if (params) {
    const variables = params.split('&').filter(filterVariable).join('&');

    return base + '?' + variables;
  }

  return base;
};

const filterVariable = (param: string) => {
  const [_, value] = param.split('=');

  // Filter out variables with the All value
  if (value === '$__all' || value === 'All') {
    return false;
  }

  // Filter out variables with no value
  if (value === '' || value === 'None') {
    return false;
  }

  return true;
};
