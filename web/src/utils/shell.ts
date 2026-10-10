export const shellQuote = (s: string): string => `'${s.replace(/'/g, "'\\''")}'`
