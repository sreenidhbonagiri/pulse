import { NavLink, Outlet } from 'react-router-dom'

export function Layout() {
  return (
    <div className="shell">
      <aside className="sidebar">
        <NavLink to="/" className="brand brand-link">
          <span className="brand-mark" aria-hidden="true" />
          <div>
            <p className="brand-name">Pulse</p>
            <p className="brand-tag">Uptime monitoring</p>
          </div>
        </NavLink>
        <nav className="nav">
          <NavLink to="/dashboard">Dashboard</NavLink>
          <NavLink to="/monitors/new">Add Monitor</NavLink>
        </nav>
      </aside>
      <div className="shell-main">
        <header className="topbar">
          <NavLink to="/" className="top-link">
            Home
          </NavLink>
          <NavLink to="/dashboard" className="top-link">
            Dashboard
          </NavLink>
          <NavLink to="/monitors/new" className="top-link">
            Add Monitor
          </NavLink>
        </header>
        <main className="content">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
