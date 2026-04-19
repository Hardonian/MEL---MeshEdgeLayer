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
import { OperationalReview } from './pages/OperationalReview'
import { Incidents } from './pages/Incidents'
import { IncidentDetail } from './pages/IncidentDetail'
import { ControlActions } from './pages/ControlActions'
import { Topology } from './pages/Topology'
import { Planning } from './pages/Planning'
import { Transports } from './pages/Transports'
import { ApiProvider } from './hooks/useApi'
import { ConsoleThemeProvider } from './hooks/useConsoleThemePreference'
import { ToastProvider } from './components/ui/Toast'
import { OperatorWorkspaceFocusProvider } from './hooks/useOperatorWorkspaceFocus'

export default function App() {
  return (
    <ApiProvider>
      <OperatorWorkspaceFocusProvider>
        <ConsoleThemeProvider>
          <ToastProvider>
            <Layout>
              <Routes>
                <Route path="/" element={<Dashboard />} />
                <Route path="/status" element={<Status />} />
                <Route path="/transports" element={<Transports />} />
                <Route path="/nodes" element={<Nodes />} />
                <Route path="/nodes/:nodeId" element={<Nodes />} />
                <Route path="/topology" element={<Topology />} />
                <Route path="/planning" element={<Planning />} />
                <Route path="/messages" element={<Messages />} />
                <Route path="/privacy" element={<Privacy />} />
                <Route path="/dead-letters" element={<DeadLetters />} />
                <Route path="/incidents" element={<Incidents />} />
                <Route path="/incidents/:id" element={<IncidentDetail />} />
                <Route path="/control-actions" element={<ControlActions />} />
                <Route path="/recommendations" element={<Recommendations />} />
                <Route path="/events" element={<Events />} />
                <Route path="/settings" element={<SettingsPage />} />
                <Route path="/diagnostics" element={<Diagnostics />} />
                <Route path="/operational-review" element={<OperationalReview />} />
                <Route path="*" element={<Navigate to="/" replace />} />
              </Routes>
            </Layout>
          </ToastProvider>
        </ConsoleThemeProvider>
      </OperatorWorkspaceFocusProvider>
    </ApiProvider>
  )
}
