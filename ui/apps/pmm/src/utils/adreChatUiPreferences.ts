export const ADRE_CHAT_UI_STORAGE_KEY = 'pmm-adre-chat-ui';

export type AdreChatUiMode = 'fast' | 'investigation';

export type AdreChatUiPrefs = {
  model?: string;
  mode?: AdreChatUiMode;
};

function isAdreChatUiMode(v: unknown): v is AdreChatUiMode {
  return v === 'fast' || v === 'investigation';
}

/** Server/settings default when no stored mode: legacy `chat` matches Fast in the UI. */
export function defaultChatModeFromSettings(defaultChatMode: string | undefined): AdreChatUiMode {
  if (defaultChatMode === 'investigation') return 'investigation';
  if (defaultChatMode === 'fast' || defaultChatMode === 'chat') return 'fast';

  return 'investigation';
}

export function loadAdreChatUiPreferences(): AdreChatUiPrefs {
  try {
    const raw = localStorage.getItem(ADRE_CHAT_UI_STORAGE_KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {};

    const rec = parsed as Record<string, unknown>;
    const prefs: AdreChatUiPrefs = {};
    if (typeof rec.model === 'string' && rec.model.length > 0) prefs.model = rec.model;
    if (isAdreChatUiMode(rec.mode)) prefs.mode = rec.mode;

    return prefs;
  } catch {
    return {};
  }
}

/** Merge into existing stored prefs. Use `model: ''` or `removeModel: true` to clear a stored model id. */
export function saveAdreChatUiPreferences(
  patch: Partial<Pick<AdreChatUiPrefs, 'mode' | 'model'>> & { removeModel?: boolean }
): void {
  try {
    const prev = loadAdreChatUiPreferences();
    const next: AdreChatUiPrefs = { ...prev };
    if ('mode' in patch && patch.mode !== undefined) {
      next.mode = patch.mode;
    }
    if (patch.removeModel) {
      delete next.model;
    } else if ('model' in patch) {
      if (patch.model === undefined || patch.model === '') delete next.model;
      else next.model = patch.model;
    }
    if (next.mode === undefined && next.model === undefined) {
      localStorage.removeItem(ADRE_CHAT_UI_STORAGE_KEY);
    } else {
      const out: Record<string, string> = {};
      if (next.mode !== undefined) out.mode = next.mode;
      if (next.model !== undefined) out.model = next.model;
      localStorage.setItem(ADRE_CHAT_UI_STORAGE_KEY, JSON.stringify(out));
    }
  } catch {
    // ignore quota / private mode
  }
}
