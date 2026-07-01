import { useState, type ReactNode } from 'react'
import { Box, Button, Paper, TextField, Typography, Alert } from '@mui/material'
import { saveCredentials } from '../api/client'
import { getProfile } from '../api/profile'

interface Props {
  children: ReactNode
}

export default function LoginGate({ children }: Props) {
  const [authed, setAuthed] = useState(() => localStorage.getItem('jobauto.basicAuth') !== null)
  const [user, setUser] = useState('')
  const [pass, setPass] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [checking, setChecking] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setChecking(true)
    saveCredentials(user, pass)
    try {
      await getProfile()
      setAuthed(true)
    } catch {
      setError('Invalid credentials')
    } finally {
      setChecking(false)
    }
  }

  if (authed) return <>{children}</>

  return (
    <Box className="flex justify-center items-center min-h-screen">
      <Paper className="p-8 w-full max-w-sm" elevation={3}>
        <Typography variant="h5" className="mb-4">
          Job Automation
        </Typography>
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <TextField label="Username" value={user} onChange={(e) => setUser(e.target.value)} autoFocus />
          <TextField label="Password" type="password" value={pass} onChange={(e) => setPass(e.target.value)} />
          {error && <Alert severity="error">{error}</Alert>}
          <Button type="submit" variant="contained" disabled={checking}>
            {checking ? 'Checking…' : 'Sign in'}
          </Button>
        </form>
      </Paper>
    </Box>
  )
}
