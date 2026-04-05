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
          DEFAULT: "hsl(var(--state-success))",
          foreground: "hsl(var(--foreground))",
        },
        warning: {
          DEFAULT: "hsl(var(--state-warning))",
          foreground: "hsl(var(--foreground))",
        },
        critical: {
          DEFAULT: "hsl(var(--state-critical))",
          foreground: "hsl(var(--foreground))",
        },
        info: {
          DEFAULT: "hsl(var(--info))",
          foreground: "hsl(var(--info-foreground))",
        },
        state: {
          normal: "hsl(var(--state-normal))",
          active: "hsl(var(--state-active))",
          observed: "hsl(var(--state-observed))",
          inferred: "hsl(var(--state-inferred))",
          stale: "hsl(var(--state-stale))",
          degraded: "hsl(var(--state-degraded))",
          warning: "hsl(var(--state-warning))",
          critical: "hsl(var(--state-critical))",
          frozen: "hsl(var(--state-frozen))",
          unsupported: "hsl(var(--state-unsupported))",
          success: "hsl(var(--state-success))",
        },
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
      fontFamily: {
        sans: ['Manrope', 'system-ui', 'sans-serif'],
        mono: ['IBM Plex Mono', 'JetBrains Mono', 'Menlo', 'monospace'],
        inter: ['Manrope', 'system-ui', 'sans-serif'],
        outfit: ['Outfit', 'system-ui', 'sans-serif'],
      },
      spacing: {
        '18': '4.5rem',
        '88': '22rem',
      },
      maxWidth: {
        '8xl': '88rem',
      },
      boxShadow: {
        panel: '0 18px 42px -26px hsl(var(--shell-shadow) / 0.55), 0 10px 18px -14px hsl(var(--shell-shadow) / 0.35)',
        float: '0 22px 56px -28px hsl(var(--shell-shadow) / 0.65), 0 14px 28px -20px hsl(var(--shell-shadow) / 0.45)',
        chrome: '0 18px 48px -32px hsl(var(--shell-shadow) / 0.72)',
        inset: 'inset 0 1px 0 hsl(var(--foreground) / 0.05)',
      },
    },
  },
  plugins: [],
}
