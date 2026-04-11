/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        bore: {
          bg: "#1a1a2e",
          surface: "#232338",
          card: "#2a2a40",
          border: "#3a3a50",
          sidebar: "#1e1e32",
          accent: "#4a9eff",
          "accent-hover": "#3a8eef",
          text: "#e0e0e0",
          "text-muted": "#888",
          "text-dim": "#666",
          active: "#22c55e",
          error: "#ef4444",
          warning: "#f59e0b",
          stopped: "#6b7280",
          connecting: "#3b82f6",
        },
      },
      fontFamily: {
        mono: [
          "JetBrains Mono",
          "Fira Code",
          "SF Mono",
          "Menlo",
          "monospace",
        ],
      },
    },
  },
  plugins: [],
};
