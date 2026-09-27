import 'vuetify/styles'
import '@mdi/font/css/materialdesignicons.css'
import { createVuetify } from 'vuetify'
import { darkColors, lightColors, readStoredTheme } from '../styles/tokens'

// CCMAI-UX-000: colors come from the design tokens; the theme choice persists per browser.
export default createVuetify({
  theme: {
    defaultTheme: readStoredTheme(),
    themes: {
      light: {
        dark: false,
        colors: lightColors,
        variables: {
          'border-color': '#1B1F24',
          'border-opacity': 0.14,
        },
      },
      dark: {
        dark: true,
        colors: darkColors,
        variables: {
          'border-color': '#E7EAF0',
          'border-opacity': 0.16,
        },
      },
    },
  },
  defaults: {
    // Flat cards with a border; dialogs and menus keep a shadow so they read as layers.
    VCard: {
      rounded: 'lg',
      elevation: 0,
      border: true,
    },
    VDialog: {
      VCard: { elevation: 8, border: false },
    },
    VMenu: {
      VCard: { elevation: 6, border: false },
    },
    VBtn: {
      rounded: 'lg',
    },
    VChip: {
      rounded: 'lg',
    },
    VTextField: {
      variant: 'outlined',
      density: 'comfortable',
    },
    VSelect: {
      variant: 'outlined',
      density: 'comfortable',
    },
  },
})
