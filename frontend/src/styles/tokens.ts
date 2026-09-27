// CCMAI-UX-000: design tokens from docs/decisions/UI_DESIGN_DIRECTION_2026-09-27.md §2.1.
// Status colors are text/background pairs checked for WCAG AA by src/__tests__/theme.spec.ts.

export type ThemeColors = Record<string, string>

export const lightColors: ThemeColors = {
  primary: '#3342A8',
  secondary: '#5B6470',
  background: '#F5F6F8',
  surface: '#FFFFFF',
  'on-background': '#1B1F24',
  'on-surface': '#1B1F24',
  success: '#1E6B34',
  error: '#B42318',
  warning: '#7A4100',
  info: '#2F5D9E',
  pass: '#1E6B34',
  'pass-bg': '#E4F4EA',
  fail: '#A3261B',
  'fail-bg': '#FDECEA',
  skip: '#4F5661',
  'skip-bg': '#ECEEF1',
  'src-changed': '#7A4100',
  'src-changed-bg': '#FFF1DC',
  'src-unavailable': '#5B3A99',
  'src-unavailable-bg': '#F1EBFB',
  'src-legacy': '#5B6470',
  danger: '#B42318',
  border: '#E3E6EB',
  'text-muted': '#5B6470',
  // Text on filled semantic colors. Set explicitly: Vuetify's automatic choice can pick
  // white on the light dark-theme fills, which fails contrast.
  'on-primary': '#FFFFFF',
  'on-secondary': '#FFFFFF',
  'on-success': '#FFFFFF',
  'on-error': '#FFFFFF',
  'on-warning': '#FFFFFF',
  'on-info': '#FFFFFF',
  'on-danger': '#FFFFFF',
}

export const darkColors: ThemeColors = {
  primary: '#9AA6F0',
  secondary: '#A3ABB8',
  background: '#0F1217',
  surface: '#171B22',
  'on-background': '#E7EAF0',
  'on-surface': '#E7EAF0',
  success: '#7FD39A',
  error: '#F2998F',
  warning: '#F5BE7A',
  info: '#9CC0F5',
  pass: '#7FD39A',
  'pass-bg': '#15301F',
  fail: '#F2998F',
  'fail-bg': '#3A1B18',
  skip: '#B4BAC4',
  'skip-bg': '#242A33',
  'src-changed': '#F5BE7A',
  'src-changed-bg': '#3A2A12',
  'src-unavailable': '#C3A9F2',
  'src-unavailable-bg': '#2A2140',
  'src-legacy': '#A3ABB8',
  danger: '#C9372C',
  border: '#2A303A',
  'text-muted': '#A3ABB8',
  'on-primary': '#0F1217',
  'on-secondary': '#0F1217',
  'on-success': '#0F1217',
  'on-error': '#0F1217',
  'on-warning': '#0F1217',
  'on-info': '#0F1217',
  'on-danger': '#FFFFFF',
}

// Pairs that must stay readable: [foreground token, background token, minimum ratio].
// '#FFFFFF' is literal white text on a filled button.
export const CONTRAST_PAIRS: [string, string, number][] = [
  ['on-surface', 'surface', 4.5],
  ['on-background', 'background', 4.5],
  ['text-muted', 'surface', 4.5],
  ['text-muted', 'background', 4.5],
  ['pass', 'pass-bg', 4.5],
  ['fail', 'fail-bg', 4.5],
  ['skip', 'skip-bg', 4.5],
  ['src-changed', 'src-changed-bg', 4.5],
  ['src-unavailable', 'src-unavailable-bg', 4.5],
  // Source-status chips are outlined on the card surface.
  ['src-changed', 'surface', 4.5],
  ['src-unavailable', 'surface', 4.5],
  ['src-legacy', 'surface', 4.5],
  ['src-legacy', 'background', 4.5],
  ['primary', 'surface', 4.5],
  // Filled buttons, app bars and flat chips.
  ['on-primary', 'primary', 4.5],
  ['on-secondary', 'secondary', 4.5],
  ['on-success', 'success', 4.5],
  ['on-error', 'error', 4.5],
  ['on-warning', 'warning', 4.5],
  ['on-info', 'info', 4.5],
  ['on-danger', 'danger', 4.5],
]

// Filled buttons whose text is white in the given theme.
export const WHITE_TEXT_ON: Record<'light' | 'dark', string[]> = {
  light: ['primary', 'danger'],
  dark: ['danger'],
}

export const THEME_STORAGE_KEY = 'ccma_theme'

export function readStoredTheme(): 'light' | 'dark' {
  try {
    return localStorage.getItem(THEME_STORAGE_KEY) === 'dark' ? 'dark' : 'light'
  } catch {
    return 'light'
  }
}

export function storeTheme(name: string) {
  try {
    localStorage.setItem(THEME_STORAGE_KEY, name === 'dark' ? 'dark' : 'light')
  } catch {
    // Storage can be unavailable (private mode); the toggle still works for this page.
  }
}

// WCAG 2.x relative luminance and contrast ratio.
function channel(v: number) {
  const s = v / 255
  return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4)
}

export function luminance(hex: string) {
  const h = hex.replace('#', '')
  const r = parseInt(h.slice(0, 2), 16)
  const g = parseInt(h.slice(2, 4), 16)
  const b = parseInt(h.slice(4, 6), 16)
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)
}

export function contrastRatio(a: string, b: string) {
  const la = luminance(a)
  const lb = luminance(b)
  return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05)
}
