const INTENT_NOTE_PREFIX = "Strategy intent:\n";

export function extractIntentFromNotes(notes: string | null | undefined): string {
  if (!notes) return "";
  if (notes.startsWith(INTENT_NOTE_PREFIX)) {
    const rest = notes.slice(INTENT_NOTE_PREFIX.length);
    const cut = rest.indexOf("\n\n---\n");
    return (cut >= 0 ? rest.slice(0, cut) : rest).trim();
  }
  return "";
}

export function mergeIntentIntoNotes(intent: string, previousNotes: string): string {
  const trimmed = intent.trim();
  let tail = "";
  if (previousNotes.startsWith(INTENT_NOTE_PREFIX)) {
    const idx = previousNotes.indexOf("\n\n---\n");
    if (idx >= 0) {
      tail = previousNotes.slice(idx + "\n\n---\n".length).trim();
    }
  } else if (previousNotes.trim()) {
    tail = previousNotes.trim();
  }
  const block = `${INTENT_NOTE_PREFIX}${trimmed}`;
  if (tail) return `${block}\n\n---\n${tail}`;
  return block;
}
