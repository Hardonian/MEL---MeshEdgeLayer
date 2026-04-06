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
          foreground: "hsl(var(--success) / 0.15)",
        },
        warning: {
          DEFAULT: "hsl(var(--warning))",
          foreground: "hsl(var(--warning) / 0.15)",
        },
        critical: {
          DEFAULT: "hsl(var(--critical))",
          foreground: "hsl(var(--critical) / 0.15)",
        },
        info: {
          DEFAULT: "hsl(var(--info))",
          foreground: "hsl(var(--info-foreground))",
        },
        // MEL semantic
        observed: "hsl(var(--observed))",
        inferred: "hsl(var(--inferred))",
        stale: "hsl(var(--stale))",
        frozen: "hsl(var(--frozen))",
        active: "hsl(var(--active))",
        degraded: "hsl(var(--degraded))",
        unsupported: "hsl(var(--unsupported))",
        // Neon system
        neon: {
          DEFAULT: "hsl(var(--neon))",
          alt: "hsl(var(--neon-alt))",
          hot: "hsl(var(--neon-hot))",
          warn: "hsl(var(--neon-warn))",
        },
        chrome: {
          bg: "hsl(var(--chrome-bg))",
          border: "hsl(var(--chrome-border))",
        },
        panel: {
          DEFAULT: "hsl(var(--panel))",
          muted: "hsl(var(--panel-muted))",
          strong: "hsl(var(--panel-strong))",
          border: "hsl(var(--panel-border))",
        },
        /** Semantic operator states — use for borders/bg/text, not arbitrary accents */
        signal: {
          live: "hsl(var(--signal-live))",
          observed: "hsl(var(--signal-observed))",
          inferred: "hsl(var(--signal-inferred))",
          stale: "hsl(var(--signal-stale))",
          degraded: "hsl(var(--signal-degraded))",
          critical: "hsl(var(--signal-critical))",
          frozen: "hsl(var(--signal-frozen))",
          unsupported: "hsl(var(--signal-unsupported))",
          complete: "hsl(var(--signal-complete))",
        },
      },
      borderRadius: {
        lg: "calc(var(--radius) + 2px)",
        md: "var(--radius)",
        sm: "calc(var(--radius) - 1px)",
      },
      fontFamily: {
        sans: [
          'IBM Plex Sans',
          'system-ui',
          'Segoe UI',
          'Helvetica Neue',
          'Arial',
          'sans-serif',
        ],
        mono: ['JetBrains Mono', 'IBM Plex Mono', 'Menlo', 'Monaco', 'monospace'],
        display: ['IBM Plex Sans', 'system-ui', 'sans-serif'],
        /** Dense metrics, IDs, timestamps — use with text-mel-* for legibility */
        data: ['JetBrains Mono', 'IBM Plex Mono', 'Menlo', 'Monaco', 'monospace'],
      },
      fontSize: {
        'mel-xs': ['10px', { lineHeight: '1.35', letterSpacing: '0.04em' }],
        'mel-sm': ['11px', { lineHeight: '1.45', letterSpacing: '0.02em' }],
        'mel-base': ['13px', { lineHeight: '1.55', letterSpacing: '0.015em' }],
        'mel-label': ['10px', { lineHeight: '1.25', letterSpacing: '0.12em', fontWeight: '700' }],
        'mel-metric': ['1.2rem', { lineHeight: '1.12', letterSpacing: '-0.02em', fontWeight: '700' }],
        'mel-metric-lg': ['1.65rem', { lineHeight: '1.1', letterSpacing: '-0.03em', fontWeight: '700' }],
      },
      spacing: {
        '18': '4.5rem',
        '88': '22rem',
      },
      maxWidth: {
        '8xl': '88rem',
      },
      boxShadow: {
        panel: 'none',
        float: '0 8px 24px hsl(var(--shell-shadow) / 0.25)',
        chrome: 'none',
        inset: 'none',
        glow: '0 0 0 1px hsl(var(--border) / 0.7)',
        'glow-strong': '0 10px 30px hsl(var(--shell-shadow) / 0.35)',
        'glow-hot': '0 0 0 1px hsl(var(--neon-hot) / 0.35)',
        'neon-border': 'inset 0 0 0 1px hsl(var(--neon) / 0.15)',
      },
    },
  },
  plugins: [],
}
