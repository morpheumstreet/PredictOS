/** Matches Wallet Tracking wallet address field (WalletTrackingTerminal) */
const FIELD_BASE =
  "rounded-lg border border-border bg-secondary/50 text-sm transition-all placeholder:text-muted-foreground/50 hover:border-primary/50 focus:outline-none focus:border-primary disabled:opacity-50 disabled:cursor-not-allowed";

export const PLACEHOLDER_HINT = `Placeholders: {event_title} {event_slug} {description} {resolution_source} — markets also: {market_question} {market_slug} {market_id} {event_id}`;

export const INPUT_FIELD = `w-full px-4 py-3 font-mono ${FIELD_BASE}`;
export const TEXTAREA_FIELD = `w-full px-4 py-3 ${FIELD_BASE} resize-y leading-relaxed`;
export const TEXTAREA_MONO = `w-full px-4 py-3 font-mono ${FIELD_BASE} resize-y`;
export const INPUT_READONLY =
  "w-full cursor-default rounded-lg border border-border bg-secondary/40 px-4 py-3 font-mono text-sm opacity-90 focus:outline-none";
export const CHECKBOX_FIELD =
  "h-4 w-4 cursor-pointer rounded border-border bg-secondary/50 accent-primary focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 focus-visible:ring-offset-2 focus-visible:ring-offset-card";

export const MODAL_BTN_FOCUS =
  "focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/45 focus-visible:ring-offset-2 focus-visible:ring-offset-card";

export const BTN_PRIMARY_ACTION = `flex items-center gap-2 px-4 py-2 rounded-lg bg-primary/20 border border-primary/50 text-primary text-sm font-mono hover:bg-primary/30 disabled:opacity-50 disabled:focus-visible:ring-0 ${MODAL_BTN_FOCUS}`;
export const BTN_HEADER_NEW = `flex items-center gap-2 px-4 py-2 rounded-lg bg-primary/15 border border-primary/40 text-primary hover:bg-primary/25 transition-colors text-sm font-mono ${MODAL_BTN_FOCUS}`;
export const BTN_GHOST = `px-4 py-2 rounded-lg border border-border text-sm hover:bg-secondary ${MODAL_BTN_FOCUS}`;
export const BTN_ICON_EDIT = `p-2 rounded-md hover:bg-secondary border border-transparent hover:border-border ${MODAL_BTN_FOCUS}`;
export const BTN_ICON_DELETE = `p-2 rounded-md hover:bg-destructive/15 border border-transparent hover:border-destructive/40 text-destructive ${MODAL_BTN_FOCUS}`;
