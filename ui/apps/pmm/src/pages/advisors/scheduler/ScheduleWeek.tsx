import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import Stack from '@mui/material/Stack';
import { alpha, type Theme } from '@mui/material/styles';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { type FC } from 'react';
import { type AdvisorSchedule } from 'types/advisors.types';
import { MONO_FONT, Swatch } from './AdvisorScheduler.components';
import { Messages } from './AdvisorScheduler.messages';
import {
  assignLanes,
  formatLocalWeekdayTime,
  formatUtcDay,
  formatUtcTime,
  isUtcViewer,
  occurrencesBetween,
  WEEKDAYS_SHORT,
} from './AdvisorScheduler.utils';

const DAY_MS = 86_400_000;
const AXIS_HOURS = [0, 3, 6, 9, 12, 15, 18, 21, 24];
// markers closer than this share of a day would overlap, so they stack
const COLLISION = 0.06;
const ROW_HEIGHT = 34;
const LANE_HEIGHT = 17;

const todayBackground = (theme: Theme) =>
  alpha(theme.palette.primary.main, 0.09);

interface Mark {
  schedule: AdvisorSchedule;
  at: Date;
  // position within the day, 0 to 1
  x: number;
  // lane from the top within its group of colliding markers, and the group's lane count
  lane: number;
  lanes: number;
}

interface ScheduleWeekProps {
  schedules: AdvisorSchedule[];
  colors: Map<string, string>;
  now: Date;
  advisorsOn: boolean;
  // checks and services a run of the schedule covers, for the marker label
  coverageOf: (schedule: AdvisorSchedule) => {
    checks: number;
    services: number;
  };
  onOpen: (schedule: AdvisorSchedule) => void;
}

export const ScheduleWeek: FC<ScheduleWeekProps> = ({
  schedules,
  colors,
  now,
  advisorsOn,
  coverageOf,
  onOpen,
}) => {
  const sinceMonday = (now.getUTCDay() + 6) % 7;
  const monday = Date.UTC(
    now.getUTCFullYear(),
    now.getUTCMonth(),
    now.getUTCDate() - sinceMonday
  );
  const enabled = schedules.filter((s) => s.enabled);

  const renderBody = () => {
    if (!schedules.length || !advisorsOn) {
      return (
        <Typography
          variant="body2"
          color="text.secondary"
          sx={{ py: 2 }}
          data-testid="schedule-week-empty"
        >
          {schedules.length
            ? Messages.week.advisorsOff
            : Messages.week.noSchedules}
        </Typography>
      );
    }

    return (
      <Box sx={{ overflowX: 'auto' }}>
        <Box
          sx={{
            minWidth: 620,
            display: 'grid',
            gridTemplateColumns: '74px minmax(0, 1fr)',
          }}
        >
          <Box />
          <Box
            sx={{
              position: 'relative',
              height: 18,
              fontFamily: MONO_FONT,
              fontSize: 11,
              color: 'text.disabled',
            }}
          >
            {AXIS_HOURS.map((hour, i) => (
              <Box
                component="span"
                key={hour}
                sx={{
                  position: 'absolute',
                  left: `${(hour / 24) * 100}%`,
                  transform:
                    i === 0
                      ? 'none'
                      : i === AXIS_HOURS.length - 1
                        ? 'translateX(-100%)'
                        : 'translateX(-50%)',
                }}
              >
                {`${String(hour).padStart(2, '0')}:00`}
              </Box>
            ))}
          </Box>
          {Array.from({ length: 7 }, (_, i) => {
            const dayStart = new Date(monday + i * DAY_MS);
            const dayEnd = new Date(monday + (i + 1) * DAY_MS);
            const isToday = now >= dayStart && now < dayEnd;
            const marks: Mark[] = enabled
              .flatMap((schedule) =>
                occurrencesBetween(
                  schedule.daysOfWeek ?? [],
                  schedule.timeOfDay,
                  dayStart,
                  dayEnd
                ).map((at) => ({
                  schedule,
                  at,
                  x: (at.getTime() - dayStart.getTime()) / DAY_MS,
                  lane: 0,
                  lanes: 1,
                }))
              )
              .sort((a, b) => a.x - b.x);
            const rowHeight = Math.max(
              ROW_HEIGHT,
              assignLanes(marks, COLLISION) * LANE_HEIGHT
            );

            return [
              <Box
                key={`day-${i}`}
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  height: rowHeight,
                  borderTop: 1,
                  borderColor: 'divider',
                  fontSize: 12.5,
                  color: isToday ? 'primary.main' : 'text.secondary',
                  fontWeight: isToday ? 600 : undefined,
                  bgcolor: isToday ? todayBackground : undefined,
                }}
                data-testid={isToday ? 'schedule-week-today' : undefined}
              >
                {`${WEEKDAYS_SHORT[dayStart.getUTCDay()]} ${dayStart.getUTCDate()}`}
              </Box>,
              <Box
                key={`track-${i}`}
                sx={{
                  position: 'relative',
                  height: rowHeight,
                  borderTop: 1,
                  borderColor: 'divider',
                  backgroundImage: (theme) =>
                    `linear-gradient(to right, ${theme.palette.divider} 1px, transparent 1px)`,
                  backgroundSize: '12.5% 100%',
                  bgcolor: isToday ? todayBackground : undefined,
                }}
              >
                {isToday && (
                  <Box
                    sx={{
                      position: 'absolute',
                      top: 0,
                      bottom: 0,
                      left: `${((now.getTime() - dayStart.getTime()) / DAY_MS) * 100}%`,
                      width: 2,
                      bgcolor: 'primary.main',
                    }}
                    data-testid="schedule-week-now"
                  >
                    <Box
                      component="span"
                      sx={{
                        position: 'absolute',
                        top: -1,
                        left: 4,
                        fontSize: 10,
                        fontFamily: MONO_FONT,
                        color: 'primary.main',
                      }}
                    >
                      {Messages.week.now}
                    </Box>
                  </Box>
                )}
                {marks.map((mark) => {
                  const color = colors.get(mark.schedule.scheduleId) ?? '';
                  const coverage = coverageOf(mark.schedule);
                  const label = Messages.week.mark(
                    mark.schedule.name,
                    `${formatUtcDay(mark.at)} ${formatUtcTime(mark.at)} UTC`,
                    isUtcViewer(mark.at) ? '' : formatLocalWeekdayTime(mark.at),
                    coverage.checks,
                    coverage.services
                  );
                  return (
                    <Tooltip
                      key={`${mark.schedule.scheduleId}-${mark.at.getTime()}`}
                      title={label}
                      arrow
                    >
                      <ButtonBase
                        aria-label={label}
                        onClick={() => onOpen(mark.schedule)}
                        data-testid={`schedule-mark-${mark.schedule.scheduleId}`}
                        sx={{
                          position: 'absolute',
                          left: `${mark.x * 100}%`,
                          top: `${((mark.lane + 0.5) / mark.lanes) * 100}%`,
                          transform: 'translate(-50%, -50%)',
                          height: mark.lanes > 1 ? 16 : 20,
                          px: '7px',
                          borderRadius: '10px',
                          border: `1.5px solid ${color}`,
                          bgcolor: color,
                          color: 'background.paper',
                          fontFamily: MONO_FONT,
                          fontSize: mark.lanes > 1 ? 10 : 11,
                          fontWeight: 500,
                          whiteSpace: 'nowrap',
                          opacity: mark.at < now ? 0.4 : 1,
                        }}
                      >
                        {formatUtcTime(mark.at)}
                      </ButtonBase>
                    </Tooltip>
                  );
                })}
              </Box>,
            ];
          })}
        </Box>
      </Box>
    );
  };

  return (
    <Box
      component="section"
      aria-labelledby="advisor-schedule-week-title"
      sx={{
        border: 1,
        borderColor: 'divider',
        borderRadius: 2,
        px: 2,
        pt: 1.75,
        pb: 1.25,
        display: 'flex',
        flexDirection: 'column',
        gap: 1.25,
      }}
      data-testid="schedule-week"
    >
      <Stack
        direction="row"
        flexWrap="wrap"
        justifyContent="space-between"
        alignItems="baseline"
        gap={1}
      >
        <Typography
          id="advisor-schedule-week-title"
          variant="subtitle1"
          fontWeight={600}
        >
          {Messages.week.title}
        </Typography>
        <Stack
          direction="row"
          flexWrap="wrap"
          columnGap={1.75}
          rowGap={0.75}
          sx={{ fontSize: 12.5, color: 'text.secondary' }}
        >
          {schedules.map((s) => (
            <Stack
              key={s.scheduleId}
              direction="row"
              alignItems="center"
              gap={0.75}
              component="span"
            >
              <Swatch
                color={colors.get(s.scheduleId) ?? ''}
                paused={!s.enabled}
              />
              {s.enabled ? s.name : `${s.name} ${Messages.week.paused}`}
            </Stack>
          ))}
        </Stack>
      </Stack>
      {renderBody()}
    </Box>
  );
};
