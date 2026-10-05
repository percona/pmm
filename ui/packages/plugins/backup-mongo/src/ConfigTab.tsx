/**
 * The Configuration tab: current state first, change history second.
 *
 * The tab used to open straight onto the framework's task table -- a log of
 * configuration *attempts*, which is the right surface for Backups and Restores and
 * the wrong question for Configuration. The status panel answers "what is this
 * cluster's backup state" from metrics; the table stays below it as history, which
 * is what it was always good for: who submitted a change, what payload, and when.
 */
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { SchemaDrivenPlugin, type RenderFieldOverride } from '@sep/framework';
import { useLocation } from 'react-router-dom';

import { PbmConfigStatusPanel } from './PbmConfigStatusPanel';
import { useClusterScope } from './clusterScopedFields';
import { CONFIG_APP_NAME } from './routes';

/**
 * Whether the plugin is currently showing its list rather than a form or detail.
 *
 * Only the list is collapsible. The plugin owns its own routes -- `:entityName` for
 * the list, `:entityName/new` for create, `:entityName/:id` for detail -- so folding
 * it away unconditionally would hide the create form behind a collapsed section.
 *
 * @param pathname The current location.
 * @param routeBase The plugin's mount path.
 */
export function showsList(pathname: string, routeBase: string): boolean {
  const rest = pathname.startsWith(routeBase)
    ? pathname.slice(routeBase.length)
    : pathname;
  const segments = rest.split('/').filter(Boolean);
  // '' -> redirects to the entity; '<entity>' -> the list. Anything deeper is a
  // form or a detail page.
  return segments.length <= 1;
}

/**
 * @param basePath The app's mount path, from which the config route is derived.
 */
export function ConfigTab({
  basePath,
  renderField,
}: {
  basePath: string;
  renderField: RenderFieldOverride;
}) {
  const { pathname } = useLocation();
  const { cluster } = useClusterScope();
  const routeBase = `${basePath}/config`;

  const plugin = (
    <SchemaDrivenPlugin
      pluginName={CONFIG_APP_NAME}
      routeBase={routeBase}
      renderField={renderField}
      // The tabs above already offer Configuration, Backups and Restores; the
      // bar this plugin derives from `related_apps` is a second control to the
      // same three places, rendered below the switcher that scopes them.
      hideRelatedAppTabs
    />
  );

  if (!showsList(pathname, routeBase)) {
    return (
      <>
        <PbmConfigStatusPanel cluster={cluster} />
        {plugin}
      </>
    );
  }

  return (
    <>
      <PbmConfigStatusPanel cluster={cluster} />
      <Accordion defaultExpanded={false} disableGutters>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2">
            Configuration history — who changed what, and when
          </Typography>
        </AccordionSummary>
        <AccordionDetails>{plugin}</AccordionDetails>
      </Accordion>
    </>
  );
}
