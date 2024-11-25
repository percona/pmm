import { FC, PropsWithChildren, useEffect } from 'react';
import { Location, useNavigate, useSearchParams } from 'react-router-dom';

export const MessagesProvider: FC<PropsWithChildren> = ({ children }) => {
  const navigate = useNavigate();
  const [, setSearchParams] = useSearchParams();

  const handleLocationChange = (location: Location) => {
    console.log('LOCATION_CHANGE', location);

    if (
      location.pathname.endsWith('/settings/metrics-resolution') ||
      location.pathname.includes('/pmm-ui')
    ) {
      navigate(location.pathname);
      return;
    }

    // http://localhost/graph/d/pmm-qan/pmm-query-analytics?from=now-12h&to=now&var-interval=$__auto_interval_interval&var-environment=All&var-node_name=All&var-service_name=All&var-database=All&var-work_mem=4194304&var-version=14.13&var-max_connections=100&var-shared_buffers=134217728&var-wal_buffers=4194304&var-wal_segment_size=16777216&var-maintenance_work_mem=67108864&var-block_size=8192&var-checkpoint_segments=&var-checkpoint_timeout=300&var-default_statistics_target=100&var-seq_page_cost=1&var-random_page_cost=4&var-effective_cache_size=4294967296&var-effective_io_concurrency=1&var-fsync=0&var-autovacuum=1&var-autovacuum_analyze_scale_factor=0.1&var-autovacuum_analyze_threshold=50&var-autovacuum_vacuum_scale_factor=0.2&var-autovacuum_vacuum_threshold=50&var-autovacuum_vacuum_cost_limit=1732290832000&var-autovacuum_vacuum_cost_delay=0.002&var-autovacuum_max_workers=3&var-autovacuum_naptime=60&var-autovacuum_freeze_max_age=200000000&var-logging_collector=0&var-log_min_duration_statement=1732290832000&var-log_duration=0&var-log_lock_waits=0&var-max_wal_senders=10&var-max_wal_size=1073741824&var-min_wal_size=83886080&var-wal_compression=0&var-max_worker_processes=8&var-max_parallel_workers_per_gather=2&var-max_parallel_workers=2&var-autovacuum_work_mem=1732290832000&var-autovacuum_multixact_freeze_max_age=400000000&var-cluster=All&var-replication_set=All&var-node_type=All&var-service_type=All&var-username=All&var-schema=All
    if (location.pathname.includes('pmm-query-analytics')) {
      navigate('/query-analytics?' + location.search);
      return;
    }

    console.log({
      pmm: window.location.pathname,
      grafana: location.pathname,
    });

    if (window.location.pathname.includes(location.pathname)) {
      setSearchParams(location.search);
      console.log('matches');
    }
  };

  useEffect(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const onMessageReceived = (e: any) => {
      if (e.data && e.data.type === 'LOCATION_CHANGE') {
        handleLocationChange(e.data.data.location);
      }
    };

    console.log('pmm', 'messager');
    window.addEventListener('message', onMessageReceived);

    return () => {
      window.removeEventListener('message', onMessageReceived);
    };
  }, []);

  return <>{children}</>;
};
