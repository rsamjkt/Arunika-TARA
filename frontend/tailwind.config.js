/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        blue: {
          DEFAULT: '#1B4FD8',
          light: '#EEF2FF',
          dark: '#1E40AF',
          hover: '#1648C5',
        },
        success: { DEFAULT: '#065F46', bg: '#ECFDF5', border: '#6EE7B7' },
        warning: { DEFAULT: '#92400E', bg: '#FFFBEB', border: '#FCD34D' },
        danger: { DEFAULT: '#991B1B', bg: '#FEF2F2', border: '#FCA5A5' },
        surface: '#FFFFFF',
        bg: '#F5F6F8',
        border: '#E4E6EA',
        'border-strong': '#CDD1D9',
        text: {
          primary: '#0E1117',
          secondary: '#4B5563',
          muted: '#9CA3AF',
        },
      },
      borderRadius: {
        kiosk: '14px',
        card: '12px',
        btn: '10px',
        tag: '9999px',
      },
      fontSize: {
        'kiosk-hero': ['clamp(16px,2.5vw,22px)', { lineHeight: '1.3' }],
        'kiosk-title': ['clamp(13px,2vw,17px)', { lineHeight: '1.4' }],
        'kiosk-body': ['clamp(11px,1.6vw,14px)', { lineHeight: '1.6' }],
        'kiosk-label': ['clamp(10px,1.3vw,12px)', { lineHeight: '1.5' }],
        'kiosk-micro': ['clamp(9px,1.1vw,11px)', { lineHeight: '1.4' }],
        'kiosk-number': ['clamp(44px,7vw,60px)', { lineHeight: '1' }],
      },
    },
  },
  plugins: [],
}
