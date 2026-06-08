/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        surface: {
          base: '#ffffff',
          sidebar: '#F5F5F0',
          hover: '#f5f5f5',
          border: '#E5E5E0',
        },
        accent: '#10b981',
        warning: '#f59e0b',
        danger: '#ef4444',
        nav: {
          active: '#E8EDE8',
        },
      },
      fontFamily: {
        sans: ['Inter', 'Noto Sans SC', 'system-ui', '-apple-system', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
