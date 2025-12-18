import { DataLinkBuiltInVars, locationUtil, textUtil, urlUtil } from '@grafana/data';
import { config, getTemplateSrv } from '@grafana/runtime';
import { DashboardLink } from '@grafana/schema';

const getLinkUrl = (link: Partial<DashboardLink>) => {
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
    const urlWithLinks = getLinkUrl({
      url: url,
      keepTime: true,
      // Check if the DB type matches the current one used
      includeVars: checkDbType(url),
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

const checkDbType = (url: string): boolean => {
  const currentDB = window.location.pathname?.split('/')[3]?.split('-')[0];
  const targetDB = url?.split('/')[3]?.split('-')[0];

  // enable variable sharing between same db types and db type -> os/node
  return (currentDB !== undefined && currentDB === targetDB) || targetDB === 'node';
};

const cleanupVariables = (urlWithLinks: string) => {
  const [base, params] = urlWithLinks.split('?');

  if (params) {
    // remove variables which have the All value or the value is empty
    const variables = params
      .split('&')
      .filter((param) => !(param.includes('All') || param.endsWith('=')))
      .join('&');

    return base + '?' + variables;
  }

  return base;
};
