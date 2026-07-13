import { Box, Dialog, Link, Typography } from '@mui/material';
import { FC, useState, useEffect, useRef, memo } from 'react';
import {
  PANEL_IMAGE_ROOT_MARGIN,
  PLACEHOLDER_MIN_HEIGHT,
  PanelScrollRootContext,
  RENDER_IMAGE_TIMEOUT_MS,
  panelFetchQueue,
  panelImageCache,
  panelImageCacheSet,
  usePanelScrollRoot,
} from './adre-chat-markdown.utils';

/** Provider for scroll root; wrap chat message list with this when a scroll container exists. */
export const PanelScrollRootProvider = PanelScrollRootContext.Provider;

const GrafanaPanelImageInner: FC<{
  src: string;
  alt: string;
  dashboardHref: string | null;
}> = ({ src, alt, dashboardHref }) => {
  const [isZoomOpen, setIsZoomOpen] = useState(false);
  const [shouldLoad, setShouldLoad] = useState(false);
  const [state, setState] = useState<'loading' | { status: 'success'; url: string } | { status: 'error'; detail?: string }>('loading');
  const wrapperRef = useRef<HTMLDivElement>(null);
  const scrollRoot = usePanelScrollRoot();

  useEffect(() => {
    const el = wrapperRef.current;
    if (!el) return;
    const observer = new IntersectionObserver(
      (entries) => {
        for (const e of entries) {
          if (e.isIntersecting) {
            setShouldLoad(true);
            break;
          }
        }
      },
      { root: scrollRoot, rootMargin: PANEL_IMAGE_ROOT_MARGIN, threshold: 0 }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [scrollRoot]);

  useEffect(() => {
    if (!shouldLoad) return;

    const cached = panelImageCache.get(src);
    if (cached) {
      setState({ status: 'success', url: cached });
      return;
    }

    let releaseSlot: (() => void) | null = null;
    const safeReleaseSlot = () => {
      if (!releaseSlot) return;
      const r = releaseSlot;
      releaseSlot = null;
      r();
    };
    let mounted = true;
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), RENDER_IMAGE_TIMEOUT_MS);

    panelFetchQueue
      .acquire()
      .then((release) => {
        if (!mounted) {
          release();
          return null;
        }
        releaseSlot = release;
        setState('loading');
        return fetch(src, { credentials: 'include', signal: controller.signal });
      })
      .then(async (res) => {
        if (!res || !mounted) return null;
        const contentType = res.headers.get('Content-Type') ?? '';
        if (!res.ok) {
          let detail = `HTTP ${res.status}`;
          if (contentType.includes('application/json')) {
            try {
              const json = await res.json();
              if (json.error) detail += `: ${json.error}`;
            } catch { /* ignore */ }
          }
          throw new Error(detail);
        }
        if (!contentType.includes('image/')) {
          let detail = `Unexpected content type: ${contentType}`;
          if (contentType.includes('application/json')) {
            try {
              const json = await res.json();
              if (json.error) detail = json.error;
            } catch { /* ignore */ }
          }
          throw new Error(detail);
        }

        return res.blob();
      })
      .then((blob) => {
        if (!mounted || !blob) return;
        const objectUrl = URL.createObjectURL(blob);
        panelImageCacheSet(src, objectUrl);
        setState({ status: 'success', url: objectUrl });
      })
      .catch((err) => {
        if (mounted) setState({ status: 'error', detail: err instanceof Error ? err.message : undefined });
      })
      .finally(() => {
        clearTimeout(timeoutId);
        safeReleaseSlot();
      });

    return () => {
      mounted = false;
      controller.abort();
      clearTimeout(timeoutId);
      safeReleaseSlot();
    };
  }, [src, shouldLoad]);

  if (!shouldLoad) {
    return (
      <Box
        ref={wrapperRef}
        sx={{
          my: 1,
          minHeight: PLACEHOLDER_MIN_HEIGHT,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          bgcolor: 'rgba(255,255,255,0.03)',
          borderRadius: 1,
        }}
      >
        <Typography variant="body2" color="text.secondary">
          Panel will load when visible
        </Typography>
      </Box>
    );
  }

  if (state === 'loading') {
    return (
      <Box sx={{ my: 1, minHeight: 500, display: 'flex', alignItems: 'center', justifyContent: 'center', bgcolor: 'rgba(255,255,255,0.03)', borderRadius: 1 }}>
        <Typography variant="body2" color="text.secondary">
          Loading panel image…
        </Typography>
      </Box>
    );
  }
  if (state.status === 'error') {
    const friendlyDetail = state.detail && (state.detail.includes('<!DOCTYPE') || state.detail.length > 200)
      ? 'Panel render timed out — try opening in Grafana directly'
      : state.detail;
    return (
      <Box sx={{ my: 1 }}>
        <Typography variant="body2" color="text.secondary">
          Image failed to load{friendlyDetail ? ` (${friendlyDetail})` : ''}
        </Typography>
        {dashboardHref && (
          <Link
            href={dashboardHref}
            target="_blank"
            rel="noopener noreferrer"
            sx={{
              display: 'inline-block',
              mt: 0.5,
              fontSize: '0.8125rem',
              color: 'primary.light',
              '&:hover': { color: 'primary.main' },
            }}
          >
            Open in Grafana
          </Link>
        )}
      </Box>
    );
  }

  return (
    <Box sx={{ my: 1, width: '100%', minWidth: 0, maxWidth: '100%' }}>
      <Box
        component="img"
        src={state.url}
        alt={alt}
        loading="lazy"
        onClick={() => setIsZoomOpen(true)}
        sx={{
          display: 'block',
          width: '100%',
          maxWidth: '100%',
          height: 'auto',
          maxHeight: 420,
          borderRadius: 1,
          cursor: 'zoom-in',
          objectFit: 'contain',
          boxSizing: 'border-box',
        }}
      />
      <Box sx={{ mt: 0.5, display: 'flex', gap: 1.5, flexWrap: 'wrap' }}>
        <Link
          component="button"
          onClick={() => setIsZoomOpen(true)}
          sx={{
            display: 'inline-block',
            fontSize: '0.8125rem',
            color: 'primary.light',
            '&:hover': { color: 'primary.main' },
          }}
        >
          Expand image
        </Link>
        {dashboardHref && (
          <Link
            href={dashboardHref}
            target="_blank"
            rel="noopener noreferrer"
            sx={{
              display: 'inline-block',
              fontSize: '0.8125rem',
              color: 'primary.light',
              '&:hover': { color: 'primary.main' },
            }}
          >
            Open in Grafana
          </Link>
        )}
      </Box>
      <Dialog
        open={isZoomOpen}
        onClose={() => setIsZoomOpen(false)}
        maxWidth={false}
        PaperProps={{
          sx: {
            bgcolor: 'transparent',
            boxShadow: 'none',
            m: 0,
            overflow: 'visible',
          },
        }}
      >
        <Box
          component="img"
          src={state.url}
          alt={alt}
          sx={{
            maxWidth: '95vw',
            maxHeight: '90vh',
            width: 'auto',
            height: 'auto',
            display: 'block',
            borderRadius: 1,
          }}
        />
      </Dialog>
    </Box>
  );
};

export const GrafanaPanelImage = memo(GrafanaPanelImageInner);
