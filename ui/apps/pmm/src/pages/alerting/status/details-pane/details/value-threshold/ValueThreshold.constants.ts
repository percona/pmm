// Beyond ~10x over/under, the percentage stops conveying anything useful — e.g. a restart
// detector (`mysql_global_status_uptime < bool 5`) reads as "94620% over". Past this point we
// drop the percent and show only the direction word, so the value/threshold stays readable.
export const PERCENT_OFF_SCALE = 1000;
