import React, {
  FC, MouseEvent, useCallback, useEffect, useMemo, useRef, useState,
} from 'react';
import {
  Button, Spinner, useStyles, useStyles2,
} from '@grafana/ui';
import { cx } from '@emotion/css';
import { SERVICE_ID_PREFIX } from 'shared/core';
import { getStyles as getHighlightStyles } from 'shared/components/Hightlight/Highlight.styles';
import { showErrorNotification, showSuccessNotification } from 'shared/components/helpers/notification-manager';
import { stripPrefix } from '../database-models/utils';
import { QueryExampleResponseItem } from '../Details.types';
import { Messages } from '../Details.messages';
import {
  createServiceNowFromQanInsights,
  fetchQanInsights,
  fetchQanInsightsCache,
} from './AiInsights.service';
import { OVERLAY_LOADER_SIZE } from '../Details.constants';
import { renderAnalysis } from './AiInsights.markdown';
import { getStyles } from './AiInsights.styles';

const COPIED_MS = 1500;

export interface AiInsightsProps {
  queryId?: string;
  from: string;
  to: string;
  fingerprint?: string;
  examples: QueryExampleResponseItem[];
  examplesLoading: boolean;
}

const formatElapsed = (seconds: number): string => {
  if (seconds < 60) {
    return `${seconds}s`;
  }

  const m = Math.floor(seconds / 60);
  const s = seconds % 60;

  return `${m}m ${s}s`;
};

const formatTimestamp = (iso: string): string => {
  try {
    const d = new Date(iso);

    if (Number.isNaN(d.getTime())) return iso;

    return d.toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return iso;
  }
};

export const AiInsights: FC<AiInsightsProps> = ({
  queryId,
  from,
  to,
  fingerprint,
  examples,
  examplesLoading,
}) => {
  const [analysis, setAnalysis] = useState<string | null>(null);
  const [cachedAt, setCachedAt] = useState<string | null>(null);
  const [apiLoading, setApiLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [elapsed, setElapsed] = useState(0);
  const [creatingTicket, setCreatingTicket] = useState(false);
  const [ticketLabel, setTicketLabel] = useState<string | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const runOnceRef = useRef(false);
  const styles = useStyles2(getStyles);
  const highlightStyles = useStyles(getHighlightStyles);

  const firstExample = useMemo(
    () => examples?.find((e) => e.service_id && (e.example || e.explain_fingerprint)),
    [examples],
  );

  const serviceId = useMemo(
    () => (firstExample ? stripPrefix(firstExample.service_id, SERVICE_ID_PREFIX) : ''),
    [firstExample],
  );
  const queryText = useMemo(
    () => (firstExample ? (firstExample.example || firstExample.explain_fingerprint || '') : ''),
    [firstExample],
  );

  // Reset when the user selects a different query so the effect re-runs
  useEffect(() => {
    runOnceRef.current = false;
    setAnalysis(null);
    setCachedAt(null);
    setError(null);
    setTicketLabel(null);
  }, [queryId, fingerprint]);

  const startTimer = useCallback(() => {
    if (timerRef.current) clearInterval(timerRef.current);

    setElapsed(0);
    timerRef.current = setInterval(() => setElapsed((prev) => prev + 1), 1000);
  }, []);

  const stopTimer = useCallback(() => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const runAnalysis = useCallback((force: boolean) => {
    if (!serviceId.trim() || !queryText.trim()) return;

    setApiLoading(true);
    setError(null);
    startTimer();

    fetchQanInsights({
      service_id: serviceId,
      query_text: queryText,
      ...(queryId && { query_id: queryId }),
      ...(fingerprint && { fingerprint }),
      ...(from && { time_from: from }),
      ...(to && { time_to: to }),
      force,
    })
      .then((res) => {
        setAnalysis(res.analysis ?? '');
        setCachedAt(res.created_at ?? null);
        setError(null);
      })
      .catch((err: Error & { response?: { data?: { error?: string } } }) => {
        const msg = err?.response?.data?.error ?? err?.message ?? Messages.tabs.aiInsights.error;

        setError(msg);
        setAnalysis(null);
      })
      .finally(() => {
        setApiLoading(false);
        stopTimer();
      });
  }, [serviceId, queryText, queryId, fingerprint, from, to, startTimer, stopTimer]);

  useEffect(() => {
    if (runOnceRef.current) return () => {};

    if (examplesLoading) return () => {};

    if (!firstExample || !serviceId.trim() || !queryText.trim()) {
      setError(Messages.tabs.aiInsights.noExample);

      return () => {};
    }

    runOnceRef.current = true;

    // Try cache first, then fall back to a fresh analysis
    if (queryId) {
      setApiLoading(true);
      fetchQanInsightsCache(queryId, serviceId)
        .then((cached) => {
          if (cached?.analysis) {
            setAnalysis(cached.analysis);
            setCachedAt(cached.created_at ?? null);
            setApiLoading(false);
          } else {
            runAnalysis(false);
          }
        })
        .catch(() => {
          runAnalysis(false);
        });
    } else {
      runAnalysis(false);
    }

    return () => stopTimer();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [examplesLoading, firstExample, serviceId, queryText, queryId]);

  const analysisHtml = useMemo(
    () => (analysis ? renderAnalysis(analysis) : ''),
    [analysis],
  );

  // Code blocks are rendered HTML, so their copy buttons are handled here.
  const handleAnalysisClick = useCallback((e: MouseEvent<HTMLDivElement>) => {
    const button = (e.target as HTMLElement).closest('button[data-copy-code]');
    const code = button?.closest('.ai-code')?.querySelector('pre code');

    if (!button || !code) return;

    // The clipboard is missing outside secure contexts.
    Promise.resolve(navigator.clipboard?.writeText(code.textContent ?? '')).then(() => {
      button.setAttribute('data-copied', '');
      button.textContent = 'Copied';
      setTimeout(() => {
        button.removeAttribute('data-copied');
        button.textContent = 'Copy';
      }, COPIED_MS);
    }, () => undefined);
  }, []);

  const handleCopyAnalysis = useCallback(async () => {
    if (!analysis) return;

    try {
      await navigator.clipboard.writeText(analysis);
      showSuccessNotification({ message: Messages.tabs.aiInsights.copySuccess });
    } catch {
      showErrorNotification({ message: Messages.tabs.aiInsights.copyFailed });
    }
  }, [analysis]);

  const handleCreateServiceNowTicket = useCallback(async () => {
    if (!analysis || !serviceId.trim() || !queryText.trim()) return;

    setCreatingTicket(true);
    setTicketLabel(null);

    try {
      const res = await createServiceNowFromQanInsights({
        service_id: serviceId.trim(),
        query_text: queryText.trim(),
        analysis,
        ...(queryId ? { query_id: queryId } : {}),
        ...(fingerprint ? { fingerprint } : {}),
        ...(from ? { time_from: from } : {}),
        ...(to ? { time_to: to } : {}),
      });

      const label = res.ticket_number ?? res.ticket_id ?? '';

      if (label) {
        setTicketLabel(label);
      }

      showSuccessNotification({
        message: label
          ? `${Messages.tabs.aiInsights.ticketCreated}: ${label}`
          : Messages.tabs.aiInsights.ticketCreated,
      });
    } catch (err: unknown) {
      const ax = err as { response?: { data?: { error?: string; message?: string } }; message?: string };

      showErrorNotification({
        message:
          ax?.response?.data?.error
          ?? ax?.response?.data?.message
          ?? ax?.message
          ?? 'Failed to create ServiceNow ticket',
      });
    } finally {
      setCreatingTicket(false);
    }
  }, [analysis, serviceId, queryText, queryId, fingerprint, from, to]);

  if (examplesLoading || apiLoading) {
    return (
      <div className={styles.loadingContainer}>
        <Spinner size={OVERLAY_LOADER_SIZE} />
        <span className={styles.message}>
          {Messages.tabs.aiInsights.loading}
          {elapsed > 0 ? ` (${formatElapsed(elapsed)})` : ''}
        </span>
      </div>
    );
  }

  if (error) {
    return <p className={styles.error}>{error}</p>;
  }

  if (analysis) {
    return (
      <div className={styles.container}>
        <div className={styles.header}>
          <span className={styles.cachedAt}>
            {cachedAt ? `Last analyzed: ${formatTimestamp(cachedAt)}` : ''}
          </span>
          <div className={styles.headerActions}>
            <Button
              variant="secondary"
              fill="text"
              size="sm"
              icon="copy"
              onClick={() => {
                handleCopyAnalysis();
              }}
            >
              {Messages.tabs.aiInsights.copyAnalysis}
            </Button>
            <Button
              variant="secondary"
              fill="text"
              size="sm"
              icon="share-alt"
              onClick={() => {
                handleCreateServiceNowTicket();
              }}
              disabled={creatingTicket}
            >
              {creatingTicket
                ? Messages.tabs.aiInsights.creatingTicket
                : Messages.tabs.aiInsights.createServiceNowTicket}
            </Button>
            <Button
              variant="secondary"
              fill="text"
              size="sm"
              icon="sync"
              onClick={() => runAnalysis(true)}
              disabled={apiLoading}
            >
              {Messages.tabs.aiInsights.rerun}
            </Button>
          </div>
        </div>
        {ticketLabel ? (
          <div className={styles.ticketCreated}>
            {`${Messages.tabs.aiInsights.ticketCreated}: ${ticketLabel}`}
          </div>
        ) : null}
        {/* Clicks are handled for the copy buttons inside the code blocks. */}
        <div
          className={cx(styles.analysis, highlightStyles.highlightWrapper)}
          onClick={handleAnalysisClick}
          // Sanitized in renderAnalysis.
          // eslint-disable-next-line react/no-danger
          dangerouslySetInnerHTML={{ __html: analysisHtml }}
        />
      </div>
    );
  }

  return (
    <div className={styles.loadingContainer}>
      <Spinner size={OVERLAY_LOADER_SIZE} />
      <span className={styles.message}>{Messages.tabs.aiInsights.loading}</span>
    </div>
  );
};

export default AiInsights;
