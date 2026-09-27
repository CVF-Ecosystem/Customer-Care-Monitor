import { describe, it, expect } from 'vitest'
import { CONTRAST_PAIRS, WHITE_TEXT_ON, contrastRatio, darkColors, lightColors } from '../styles/tokens'

// CCMAI-UX-000: every status text/background pair meets WCAG AA in both themes.
describe('theme tokens', () => {
  const themes = { light: lightColors, dark: darkColors } as const

  it('defines the same token names in light and dark', () => {
    expect(Object.keys(darkColors).sort()).toEqual(Object.keys(lightColors).sort())
  })

  for (const [name, colors] of Object.entries(themes)) {
    for (const [fg, bg, min] of CONTRAST_PAIRS) {
      it(`${name}: ${fg} on ${bg} >= ${min}:1`, () => {
        expect(colors[fg], fg).toBeTruthy()
        expect(colors[bg], bg).toBeTruthy()
        expect(contrastRatio(colors[fg], colors[bg])).toBeGreaterThanOrEqual(min)
      })
    }
    for (const token of WHITE_TEXT_ON[name as 'light' | 'dark']) {
      it(`${name}: white text on ${token} >= 4.5:1`, () => {
        expect(contrastRatio('#FFFFFF', colors[token])).toBeGreaterThanOrEqual(4.5)
      })
    }
  }

  it('matches the decision document primary and danger values', () => {
    expect(lightColors.primary).toBe('#3342A8')
    expect(darkColors.danger).toBe('#C9372C')
    expect(contrastRatio('#FFFFFF', '#3342A8')).toBeCloseTo(8.45, 1)
  })
})
