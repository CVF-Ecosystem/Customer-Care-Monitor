import { describe, it, expect } from 'vitest'
import layout from '../layouts/DefaultLayout.vue?raw'
import { contrastRatio, darkColors, lightColors } from '../styles/tokens'

// CCMAI-UX-000 repair UX000-R1: the layout avatars take their initials color from the
// theme's on-color for their fill, so the text stays readable in light and dark.

describe('DefaultLayout avatars', () => {
  it('never force white initials; each avatar uses the on-color of its fill', () => {
    const avatars = [...layout.matchAll(/<v-avatar[^>]*color="(\w+)"[^>]*>\s*<span class="([^"]+)"/g)]
    expect(avatars).toHaveLength(3)
    for (const [, fill, spanClass] of avatars) {
      expect(spanClass.split(/\s+/)).toContain(`on-${fill}`)
      expect(spanClass).not.toContain('text-white')
    }
  })

  for (const [name, colors] of [['light', lightColors], ['dark', darkColors]] as const) {
    for (const fill of ['primary', 'secondary']) {
      it(`${name}: on-${fill} on ${fill} >= 4.5:1`, () => {
        expect(contrastRatio(colors[`on-${fill}`], colors[fill])).toBeGreaterThanOrEqual(4.5)
      })
    }
  }

  it('white on the dark-theme primary would fail, which is why the class changed', () => {
    expect(contrastRatio('#FFFFFF', darkColors.primary)).toBeLessThan(4.5)
  })
})
