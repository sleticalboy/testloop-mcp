export function formatText(mode: string, prefix: string, text: string): string {
  if (mode === 'short') {
    return prefix;
  }
  return prefix + text;
}
