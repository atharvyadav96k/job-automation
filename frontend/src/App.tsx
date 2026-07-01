import { useState } from 'react'
import { AppBar, CssBaseline, Tab, Tabs, ThemeProvider, Toolbar, createTheme } from '@mui/material'
import LoginGate from './components/LoginGate'
import ProfilePage from './pages/ProfilePage'
import JobsListPage from './pages/JobsListPage'
import JobDetailPage from './pages/JobDetailPage'

const theme = createTheme()

type View = { page: 'profile' } | { page: 'jobs' } | { page: 'job-detail'; jobId: number }

function App() {
  const [view, setView] = useState<View>({ page: 'jobs' })

  const topTab = view.page === 'profile' ? 'profile' : 'jobs'

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <LoginGate>
        <AppBar position="static" color="default" elevation={1}>
          <Toolbar>
            <Tabs value={topTab} onChange={(_, v) => setView(v === 'profile' ? { page: 'profile' } : { page: 'jobs' })}>
              <Tab value="jobs" label="Jobs" />
              <Tab value="profile" label="Profile" />
            </Tabs>
          </Toolbar>
        </AppBar>

        {view.page === 'profile' && <ProfilePage />}
        {view.page === 'jobs' && <JobsListPage onOpenJob={(id) => setView({ page: 'job-detail', jobId: id })} />}
        {view.page === 'job-detail' && (
          <JobDetailPage jobId={view.jobId} onBack={() => setView({ page: 'jobs' })} />
        )}
      </LoginGate>
    </ThemeProvider>
  )
}

export default App
