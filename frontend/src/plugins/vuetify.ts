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
          surface: '#fff9f3',
          'surface-bright': '#fffdf8',
          primary: '#29564c',
          secondary: '#a46f40',
          info: '#4e7771',
          error: '#a13a33',
          success: '#2f7151',
          warning: '#a06c25',
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
