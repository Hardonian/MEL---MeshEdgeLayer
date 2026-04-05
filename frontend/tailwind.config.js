/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        secondary: {
          DEFAULT: "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))",
        },
        destructive: {
          DEFAULT: "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))",
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        accent: {
          DEFAULT: "hsl(var(--accent))",
          foreground: "hsl(var(--accent-foreground))",
        },
        popover: {
          DEFAULT: "hsl(var(--popover))",
          foreground: "hsl(var(--popover-foreground))",
        },
        card: {
          DEFAULT: "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))",
        },
        success: {
          DEFAULT: "hsl(var(--success))",
          foreground: "hsl(142, 71%, 98%)",
        },
        warning: {
          DEFAULT: "hsl(var(--warning))",
          foreground: "hsl(38, 92%, 98%)",
        },
        critical: {
          DEFAULT: "hsl(var(--critical))",
          foreground: "hsl(0, 84%, 98%)",
        },
        info: {
          DEFAULT: "hsl(var(--info))",
          foreground: "hsl(var(--info-foreground))",
        },
        // MEL semantic state colors
        observed: "hsl(var(--observed))",
        inferred: "hsl(var(--inferred))",
        stale: "hsl(var(--stale))",
        frozen: "hsl(var(--frozen))",
        active: "hsl(var(--active))",
        degraded: "hsl(var(--degraded))",
        unsupported: "hsl(var(--unsupported))",
        // Chrome
        chrome: {
          bg: "hsl(var(--chrome-bg))",
          border: "hsl(var(--chrome-border))",
        },
      },
      borderRadius: {
        lg: "calc(var(--radius) + 2px)",
        md: "var(--radius)",
        sm: "calc(var(--radius) - 2px)",
      },
      fontFamily: {
        sans: ['Manrope', 'system-ui', 'sans-serif'],
        mono: ['IBM Plex Mono', 'JetBrains Mono', 'Menlo', 'monospace'],
        display: ['Manrope', 'system-ui', 'sans-serif'],
      },
      fontSize: {
        'mel-xs': ['10px', { lineHeight: '1.4', letterSpacing: '0.04em' }],
        'mel-sm': ['11px', { lineHeight: '1.4', letterSpacing: '0.02em' }],
        'mel-base': ['13px', { lineHeight: '1.6', letterSpacing: '0.01em' }],
        'mel-label': ['10px', { lineHeight: '1.2', letterSpacing: '0.14em', fontWeight: '600' }],
        'mel-metric': ['1.5rem', { lineHeight: '1.1', letterSpacing: '-0.03em', fontWeight: '700' }],
        'mel-metric-lg': ['2rem', { lineHeight: '1.1', letterSpacing: '-0.04em', fontWeight: '700' }],
      },
      spacing: {
        '18': '4.5rem',
        '88': '22rem',
      },
      maxWidth: {
        '8xl': '88rem',
      },
      boxShadow: {
        panel: '0 1px 3px hsl(var(--shell-shadow) / 0.12)',
        float: '0 4px 16px hsl(var(--shell-shadow) / 0.2)',
        chrome: '0 1px 3px hsl(var(--shell-shadow) / 0.15)',
        inset: 'inset 0 1px 0 hsl(var(--foreground) / 0.04)',
        glow: '0 0 12px hsl(var(--primary) / 0.15)',
        'glow-strong': '0 0 20px hsl(var(--primary) / 0.25)',
      },
    },
  },
  plugins: [],
}
