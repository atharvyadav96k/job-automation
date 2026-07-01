/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {},
  },
  // MUI owns component-level base styles; Tailwind is layout/spacing only.
  corePlugins: {
    preflight: false,
  },
  plugins: [],
}

