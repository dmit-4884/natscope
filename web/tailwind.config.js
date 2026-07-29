/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      fontFamily: {
        mono: [
          'ui-monospace',
          'SFMono-Regular',
          'SF Mono',
          'Menlo',
          'Monaco',
          'Consolas',
          'Liberation Mono',
          'Courier New',
          'monospace',
        ],
      },
      colors: {
        // Semantic surface colors
        surface: {
          primary: 'white',
          secondary: '#F9FAFB',   // gray-50
          tertiary: '#F3F4F6',    // gray-100
          hover: '#E5E7EB',       // gray-200
          inverse: '#111827',     // gray-900
        },
        // Semantic content/text colors
        content: {
          primary: '#111827',     // gray-900
          secondary: '#4B5563',   // gray-600
          tertiary: '#6B7280',    // gray-500
          muted: '#9CA3AF',       // gray-400
          inverse: 'white',
        },
        // Border colors
        border: {
          DEFAULT: '#E5E7EB',     // gray-200
          strong: '#D1D5DB',      // gray-300
          focus: '#3B82F6',       // blue-500
        },
        // Accent/primary action color
        accent: {
          DEFAULT: '#2563EB',     // blue-600
          hover: '#1D4ED8',       // blue-700
          light: '#EFF6FF',       // blue-50
          muted: '#DBEAFE',       // blue-100
          text: '#1D4ED8',        // blue-700
        },
        // Status colors
        status: {
          success: {
            bg: '#ECFDF5',        // green-50
            border: '#10B981',    // green-500
            text: '#059669',      // green-600
            light: '#D1FAE5',     // green-100
          },
          warning: {
            bg: '#FFFBEB',        // amber-50
            border: '#F59E0B',    // amber-500
            text: '#D97706',      // amber-600
            light: '#FEF3C7',     // amber-100
          },
          error: {
            bg: '#FEF2F2',        // red-50
            border: '#EF4444',    // red-500
            text: '#DC2626',      // red-600
            light: '#FEE2E2',     // red-100
          },
          info: {
            bg: '#EFF6FF',        // blue-50
            border: '#3B82F6',    // blue-500
            text: '#2563EB',      // blue-600
            light: '#DBEAFE',     // blue-100
          },
        },
      },
      spacing: {
        'panel': '12px',
        'section': '16px',
        'modal': '24px',
      },
      borderRadius: {
        'button': '6px',
        'card': '8px',
        'badge': '4px',
      },
      zIndex: {
        'dropdown': 'var(--z-dropdown)',
        'modal': 'var(--z-modal)',
        'modal-nested': 'var(--z-modal-nested)',
        'toast': 'var(--z-toast)',
        'tooltip': 'var(--z-tooltip)',
      },
      boxShadow: {
        'dropdown': '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
        'modal': '0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)',
        'card': '0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px 0 rgba(0, 0, 0, 0.06)',
        'card-hover': '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
      },
      fontSize: {
        // Micro size for overlines/pills — the one legal step below text-xs.
        '2xs': ['11px', { lineHeight: '14px' }],
      },
      animation: {
        'fade-in': 'fadeIn 200ms ease-out',
        'slide-up': 'slideUp 200ms ease-out',
        'slide-in-right': 'slideInRight 200ms ease-out',
        'slide-out-right': 'slideOutRight 150ms ease-in',
        'shimmer': 'shimmer 2s infinite linear',
        'flash-green': 'flashGreen 300ms ease-out',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
        slideOutRight: {
          '0%': { opacity: '1', transform: 'translateX(0)' },
          '100%': { opacity: '0', transform: 'translateX(20px)' },
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' },
        },
        flashGreen: {
          '0%': { backgroundColor: 'rgb(187 247 208)' },
          '100%': { backgroundColor: 'transparent' },
        },
      },
    },
  },
  plugins: [],
}
