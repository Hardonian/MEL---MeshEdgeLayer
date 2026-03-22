import { Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from './components/layout/Layout'
import { Dashboard } from './pages/Dashboard'
import { Status } from './pages/Status'
import { Nodes } from './pages/Nodes'
import { Messages } from './pages/Messages'
import { Privacy } from './pages/Privacy'
import { DeadLetters } from './pages/DeadLetters'
import { Recommendations } from './pages/Recommendations'
import { Events } from './pages/Events'
import { SettingsPage } from './pages/Settings'
import { Diagnostics } from './pages/Diagnostics'
import { TopologyMap } from './pages/TopologyMap'
import { ApiProvider } from './hooks/useApi'
import { ToastProvider } from './components/ui/Toast'

export default function App() {
  return (
    <ApiProvider>
      <ToastProvider>
        <Layout>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/status" element={<Status />} />
          <Route path="/nodes" element={<Nodes />} />
          <Route path="/nodes/:nodeId" element={<Nodes />} />
          <Route path="/messages" element={<Messages />} />
          <Route path="/privacy" element={<Privacy />} />
          <Route path="/dead-letters" element={<DeadLetters />} />
          <Route path="/recommendations" element={<Recommendations />} />
          <Route path="/events" element={<Events />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="/diagnostics" element={<Diagnostics />} />
          <Route path="/topology" element={<TopologyMap />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Layout>
      </ToastProvider>
    </ApiProvider>
  )
}
