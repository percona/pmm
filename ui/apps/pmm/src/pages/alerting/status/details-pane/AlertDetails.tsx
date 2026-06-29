import { FC } from 'react';
import { Grid, Stack, Typography } from '@mui/material';
import { AlertRow } from '../AlertsPage.types';
import { Messages } from './AlertDetails.messages';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import DataPoint from 'components/details-pane/DataPoint';
import { Chip } from '@percona/percona-ui';
import { STATUS_COLOR_MAP, STATUS_LABEL_MAP } from '../AlertsPage.constants';
import { formatTriggeredAt } from '../table/AlertStatusTable.utils';
import { useTimezone } from 'hooks/utils/useTimezone';

interface Props {
  alert: AlertRow;
}

const AlertDetails: FC<Props> = ({ alert }) => {
  const timezone = useTimezone();

  return (
    <Stack spacing={3}>
      <Typography variant="h6">{Messages.details.summary}</Typography>
      <Grid container spacing={3} columns={{ xs: 4 }}>
        <DataPoint size={2} title={Messages.details.alertName}>
          {alert.alertName}
        </DataPoint>
        <DataPoint size={1} title={Messages.details.state}>
          <Chip
            label={STATUS_LABEL_MAP[alert.state]}
            color={STATUS_COLOR_MAP[alert.state]}
          />
        </DataPoint>
        <DataPoint size={1} title={Messages.details.stateDuration}>
          {alert.age}
        </DataPoint>
        <DataPoint size={1} title={Messages.details.node}>
          {alert.nodeId}
        </DataPoint>
        <DataPoint size={1} title={Messages.details.service}>
          {alert.serviceName}
        </DataPoint>
        <DataPoint size={1} title={Messages.details.triggeredAt}>
          {formatTriggeredAt(alert.activeAt, timezone)}
        </DataPoint>
        <DataPoint size={1} title={Messages.details.severity}>
          {alert.labels.severity}
        </DataPoint>
        <DataPoint size={1} title={Messages.details.valueThreshold}></DataPoint>
        <DataPoint size={1} title={Messages.details.summaryLabel}>
          {alert.annotations.summary}
        </DataPoint>
        <DataPoint size={2} title={Messages.details.description}>
          {alert.annotations.description}
        </DataPoint>
      </Grid>
      <Stack spacing={2}>
        <Typography variant="h6">{Messages.details.expression}</Typography>
        <SyntaxHighlighter language="promql" content={alert.expression} />
      </Stack>
      <Stack spacing={2}>
        <Typography variant="h6">
          {Messages.details.ruleConfiguration}
        </Typography>
        <Grid container spacing={3} columns={{ xs: 4 }}>
          <DataPoint size={1} title={Messages.details.evaluate}></DataPoint>
          <DataPoint size={1} title={Messages.details.lastEvaluated}>
            {formatTriggeredAt(alert.ruleGroup.lastEvaluation, timezone)}
          </DataPoint>
          <DataPoint size={1} title={Messages.details.lastEvaluationDuration}>
            {alert.ruleGroup.evaluationTime}
          </DataPoint>
          <DataPoint size={1} title={Messages.details.pendingPeriod}></DataPoint>
          <DataPoint size={1} title={Messages.details.keepFiringFor}></DataPoint>
          <DataPoint size={1} title={Messages.details.ruleType}></DataPoint>
          <DataPoint
            size={1}
            title={Messages.details.ruleIdentifier}
          ></DataPoint>
          <DataPoint size={1} title={Messages.details.lastUpdatedBy}></DataPoint>
          <DataPoint size={1} title={Messages.details.lastUpdated}></DataPoint>
          <DataPoint size={1} title={Messages.details.templateName}>
            {alert.labels.template_name}
          </DataPoint>
          <DataPoint size={1} title={Messages.details.folder}>
            {alert.ruleGroup.file}
          </DataPoint>
          <DataPoint size={1} title={Messages.details.ruleHealth}></DataPoint>
        </Grid>
      </Stack>
    </Stack>
  );
};

export default AlertDetails;
