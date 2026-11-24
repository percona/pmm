import Typography from '@mui/material/Typography';
import {
  ChartsTooltipCell,
  ChartsTooltipContainer,
  ChartsTooltipPaper,
  ChartsTooltipRow,
  ChartsTooltipTable,
  chartsTooltipClasses,
  useAxesTooltip,
  type ChartsTooltipContainerProps,
} from '@mui/x-charts/ChartsTooltip';

// Its published type marks every HTML attribute as required; it is a styled Typography.
const Cell = ChartsTooltipCell as unknown as typeof Typography;

const Content = () => {
  const axes = useAxesTooltip();
  if (!axes) return null;

  return (
    <ChartsTooltipPaper className={chartsTooltipClasses.paper}>
      {axes.map(({ axisId, axisFormattedValue, seriesItems }) => (
        <ChartsTooltipTable key={axisId} className={chartsTooltipClasses.table}>
          <Typography component="caption">{axisFormattedValue}</Typography>
          <tbody>
            {seriesItems.map(({ seriesId, formattedValue }) => (
              <ChartsTooltipRow
                key={seriesId}
                className={chartsTooltipClasses.row}
              >
                <Cell
                  component="td"
                  className={`${chartsTooltipClasses.valueCell} ${chartsTooltipClasses.cell}`}
                >
                  {typeof formattedValue === 'string' ? formattedValue : null}
                </Cell>
              </ChartsTooltipRow>
            ))}
          </tbody>
        </ChartsTooltipTable>
      ))}
    </ChartsTooltipPaper>
  );
};

// The default tooltip names each series in a <th>, left empty for a sparkline's unnamed series.
const SparklineTooltip = (props: ChartsTooltipContainerProps) => (
  <ChartsTooltipContainer {...props} trigger="axis">
    <Content />
  </ChartsTooltipContainer>
);

export default SparklineTooltip;
