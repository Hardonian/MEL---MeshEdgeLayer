# MEL — Frontend Architecture

**The "Glassmorphism" Mesh Observer.**

This is the web-based control plane for MEL. It provides high-fidelity, real-time observability of your Meshtastic mesh using a dual-font, glassmorphism-inspired design system.

## 🏗️ Technology Stack

1. **Framework**: [React](https://reactjs.org/) + [Vite](https://vitejs.dev/)
2. **Logic**: TypeScript
3. **Styling**: [Tailwind CSS](https://tailwindcss.com/)
4. **Icons**: [Lucide React](https://lucide.dev/)
5. **State & API**: Custom hooks (`useApi.tsx`) with automatic polling and refresh logic.

## 🎨 Design System

MEL follows a "Premium Technical" aesthetic designed to be both immersive and highly legible.

### 🏛️ Typography
- **Headings**: `Outfit` — A clean, futuristic sans-serif.
- **Data/Body**: `Inter` — Optimized for high-density information.

### 💎 Styling Tokens
- **Glassmorphism**: Navigation and headers use `backdrop-filter: blur(10px)` with semi-transparent backgrounds to maintain spatial awareness.
- **Interactive States**: Modern hover effects (`card-hover`) and micro-animations for tactile feedback.

## 🧱 Component Library

### `src/components/ui/`
Standard UI primitives:
- `Card.tsx`: The primary container for data segments.
- `Badge.tsx`: Status and category labeling.
- `AlertCard.tsx`: High-visibility alerts for system failures.
- `PageHeader.tsx`: Consistent page identity.

### `src/hooks/useApi.tsx`
The data-fetching heartbeat of the app.
- `useControlStatus()`: Real-time control plane mode and reality matrix.
- `useControlHistory()`: Audit trail of remediations.
- `useStatus()`: High-level system health snapshot.

## 🚀 Development Setup

### Prerequisite: Start the MEL Backend
The frontend expects the MEL API to be running on `:8080`.
```bash
mel serve --config mel.json
```

### Start Frontend Dev Server
```bash
cd frontend
nvm use # reads frontend/.nvmrc (Node 24.x required)
npm install
npm run dev
```

### Runtime contract
- Frontend verification targets Node `24.x` only (`>=24 <25`).
- Guard script: `frontend/scripts/require-node24.mjs`.
- If Node is not 24.x, install/lint/test/build commands fail fast with an explicit runtime-contract message.

### Start Frontend Dev Server (without nvm)
```bash
cd frontend
npm install
npm run dev
```

The app will be available at `http://localhost:5173`.

---

*MEL — Truthful Local-First Mesh Observability.*
