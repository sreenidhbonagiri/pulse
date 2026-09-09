import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { Layout } from './components/Layout'
import { DashboardPage } from './pages/Dashboard'
import { LandingPage } from './pages/Landing'
import { MonitorDetailPage } from './pages/MonitorDetail'
import { NewMonitorPage } from './pages/NewMonitor'
import './App.css'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<LandingPage />} />
        <Route element={<Layout />}>
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="monitors/new" element={<NewMonitorPage />} />
          <Route path="monitors/:id" element={<MonitorDetailPage />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
