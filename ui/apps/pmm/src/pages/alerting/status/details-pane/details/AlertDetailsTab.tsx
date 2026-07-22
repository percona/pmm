import { FC } from 'react';
import { Grid, Stack, Typography } from '@mui/material';
import { Messages } from './AlertDetailsTab.messages';
import DataPoint from 'components/details-pane/DataPoint';
import ValueThreshold from './value-threshold/ValueThreshold';
import { Chip, CodeBlock } from '@percona/percona-ui';
import { STATUS_COLOR_MAP, STATUS_LABEL_MAP } from '../../AlertsPage.constants';
import { formatTriggeredAt } from '../../table/AlertStatusTable.utils';
import NotificationsOffOutlinedIcon from '@mui/icons-material/NotificationsOffOutlined';
import { useTimezone } from 'hooks/utils/useTimezone';
import { formatDurationSeconds } from 'utils/duration.utils';
import AlertSeverityDetail from './severity/AlertSeverityDetail';
import { AlertDetailsPane } from '../AlertDetailsPane.types';

interface Props {
  details: AlertDetailsPane;
}

const AlertDetailsTab: FC<Props> = ({
  details: { expression, summary, ruleConfiguration, rawData },
}) => {
  const timezone = useTimezone();
  return (
    <Stack spacing={3}>
      <Typography variant="h6">{Messages.details.summary}</Typography>
      <Grid container spacing={3} columns={{ xs: 1, sm: 2, md: 4 }}>
        <DataPoint size={2} title={Messages.details.alertName}>
          {summary.alertName}
        </DataPoint>
        <DataPoint size={1} title={Messages.details.state}>
          {summary.silenced ? (
            <Chip
              color="info"
              label={
                <Stack direction="row" alignItems="center" gap={0.5}>
                  <NotificationsOffOutlinedIcon fontSize="small" />
                  {Messages.details.silenced}
                </Stack>
              }
            />
          ) : (
            <Chip
              label={STATUS_LABEL_MAP[summary.state]}
              color={STATUS_COLOR_MAP[summary.state]}
            />
          )}
        </DataPoint>
        <DataPoint size={1} title={Messages.details.stateDuration}>
          {summary.stateDuration}
        </DataPoint>
        <DataPoint size={1} title={Messages.details.node}>
          {summary.nodeName}
        </DataPoint>
        <DataPoint size={1} title={Messages.details.service}>
          {summary.serviceName}
        </DataPoint>
        <DataPoint size={1} title={Messages.details.triggeredAt}>
          {formatTriggeredAt(summary.triggeredAt, timezone)}
        </DataPoint>
        <DataPoint size={1} title={Messages.details.severity}>
          <AlertSeverityDetail severity={summary.severity} />
        </DataPoint>
        <DataPoint size={1} title={Messages.details.valueThreshold}>
          <ValueThreshold
            definition={ruleConfiguration.definition}
            isDefinitionLoading={ruleConfiguration.isLoading}
            labels={rawData.labels}
          />
        </DataPoint>
        <DataPoint size={1} title={Messages.details.summaryLabel}>
          {summary.summary}
        </DataPoint>
        <DataPoint size={2} title={Messages.details.description}>
          {summary.description}
        </DataPoint>
      </Grid>
      <Stack spacing={2}>
        <Typography variant="h6">{Messages.details.expression}</Typography>
        <CodeBlock language="promql" content={expression} />
      </Stack>
      <Stack spacing={2}>
        <Typography variant="h6">
          {Messages.details.ruleConfiguration}
        </Typography>
        <Grid container spacing={3} columns={{ xs: 1, sm: 2, md: 4 }}>
          <DataPoint size={1} title={Messages.details.evaluate}>
            {formatDurationSeconds(ruleConfiguration.evaluationIntervalSeconds)}
          </DataPoint>
          <DataPoint size={1} title={Messages.details.lastEvaluated}>
            {ruleConfiguration.lastEvaluation &&
              formatTriggeredAt(ruleConfiguration.lastEvaluation, timezone)}
          </DataPoint>
          <DataPoint size={1} title={Messages.details.lastEvaluationDuration}>
            {ruleConfiguration.lastEvaluationDuration !== undefined &&
              formatDurationSeconds(ruleConfiguration.lastEvaluationDuration)}
          </DataPoint>
          <DataPoint size={1} title={Messages.details.pendingPeriod}>
            {formatDurationSeconds(ruleConfiguration.pendingPeriod)}
          </DataPoint>
          <DataPoint size={1} title={Messages.details.keepFiringFor}>
            {ruleConfiguration.keepFiringFor}
          </DataPoint>
          <DataPoint size={1} title={Messages.details.ruleType}>
            {ruleConfiguration.ruleType}
          </DataPoint>
          <DataPoint size={1} title={Messages.details.ruleIdentifier}>
            {ruleConfiguration.ruleUid}
          </DataPoint>
          <DataPoint size={1} title={Messages.details.lastUpdatedBy}>
            {ruleConfiguration.lastUpdatedBy}
          </DataPoint>
          <DataPoint size={1} title={Messages.details.lastUpdated}>
            {ruleConfiguration.lastUpdated}
          </DataPoint>
          <DataPoint size={1} title={Messages.details.templateName}>
            {ruleConfiguration.templateName}
          </DataPoint>
          <DataPoint size={1} title={Messages.details.folder}>
            {ruleConfiguration.folder}
          </DataPoint>
          <DataPoint size={1} title={Messages.details.ruleHealth}>
            {ruleConfiguration.ruleHealth}
          </DataPoint>
        </Grid>
      </Stack>
    </Stack>
  );
};

export default AlertDetailsTab;
