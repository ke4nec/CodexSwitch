import 'vuetify/styles';

import { createVuetify } from 'vuetify';

export const vuetify = createVuetify({
  theme: {
    defaultTheme: 'codexswitch',
    themes: {
      codexswitch: {
        dark: false,
        colors: {
          background: '#f3eadf',
          surface: '#fffaf2',
          'surface-bright': '#fffef9',
          primary: '#204f45',
          secondary: '#99683a',
          error: '#9f2f2f',
          success: '#2d6f4b',
          warning: '#9a641e',
        },
      },
    },
  },
  defaults: {
    VBtn: {
      rounded: 'xl',
      variant: 'flat',
    },
    VTextField: {
      variant: 'outlined',
      density: 'comfortable',
    },
    VTextarea: {
      variant: 'outlined',
      density: 'comfortable',
    },
    VDialog: {
      maxWidth: 760,
    },
  },
});
