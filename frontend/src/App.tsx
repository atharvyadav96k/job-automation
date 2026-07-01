import { CssBaseline, ThemeProvider, createTheme } from '@mui/material'
import LoginGate from './components/LoginGate'
import ProfilePage from './pages/ProfilePage'

const theme = createTheme()

function App() {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <LoginGate>
        <ProfilePage />
      </LoginGate>
    </ThemeProvider>
  )
}

export default App
