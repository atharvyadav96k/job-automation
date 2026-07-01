import { useEffect, useState } from 'react'
import {
  Box,
  Button,
  Chip,
  CircularProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  Paper,
  Alert,
} from '@mui/material'
import type { JobSummary } from '../api/jobs'
import { listJobs, runDiscovery, tailorJob } from '../api/jobs'

interface Props {
  onOpenJob: (id: number) => void
}

const STATUS_COLOR: Record<string, 'default' | 'primary' | 'success' | 'warning'> = {
  new: 'default',
  scored: 'primary',
  skipped: 'warning',
  queued: 'default',
  applied: 'success',
}

export default function JobsListPage({ onOpenJob }: Props) {
  const [jobs, setJobs] = useState<JobSummary[] | null>(null)
  const [scanning, setScanning] = useState(false)
  const [tailoringId, setTailoringId] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function load() {
    setJobs(await listJobs())
  }

  useEffect(() => {
    load()
  }, [])

  async function handleScan() {
    setScanning(true)
    setError(null)
    try {
      await runDiscovery()
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Scan failed')
    } finally {
      setScanning(false)
    }
  }

  async function handleTailor(id: number) {
    setTailoringId(id)
    setError(null)
    try {
      await tailorJob(id)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Tailoring failed')
    } finally {
      setTailoringId(null)
    }
  }

  if (!jobs) {
    return (
      <Box className="flex justify-center items-center min-h-screen">
        <CircularProgress />
      </Box>
    )
  }

  return (
    <Box className="max-w-5xl mx-auto p-6">
      <Box className="flex justify-between items-center mb-6">
        <Typography variant="h4">Jobs</Typography>
        <Button variant="contained" onClick={handleScan} disabled={scanning}>
          {scanning ? 'Scanning…' : 'Scan now'}
        </Button>
      </Box>
      {error && (
        <Alert severity="error" className="mb-4">
          {error}
        </Alert>
      )}
      <TableContainer component={Paper}>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Title</TableCell>
              <TableCell>Company</TableCell>
              <TableCell>Status</TableCell>
              <TableCell align="right">Match</TableCell>
              <TableCell align="right">ATS</TableCell>
              <TableCell align="right">Versions</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {jobs.map((job) => (
              <TableRow key={job.id} hover>
                <TableCell
                  className="cursor-pointer"
                  onClick={() => onOpenJob(job.id)}
                >
                  {job.title}
                </TableCell>
                <TableCell>{job.company_name}</TableCell>
                <TableCell>
                  <Chip
                    size="small"
                    label={job.status}
                    color={STATUS_COLOR[job.status] ?? 'default'}
                  />
                  {job.approved && <Chip size="small" label="approved" color="success" className="ml-1" />}
                </TableCell>
                <TableCell align="right">{job.latest_match_score ?? '—'}</TableCell>
                <TableCell align="right">{job.latest_ats_score ?? '—'}</TableCell>
                <TableCell align="right">{job.version_count}</TableCell>
                <TableCell align="right">
                  <Button
                    size="small"
                    onClick={() => handleTailor(job.id)}
                    disabled={tailoringId === job.id}
                  >
                    {tailoringId === job.id ? 'Tailoring…' : job.version_count > 0 ? 'Re-tailor' : 'Tailor'}
                  </Button>
                </TableCell>
              </TableRow>
            ))}
            {jobs.length === 0 && (
              <TableRow>
                <TableCell colSpan={7} className="text-center text-gray-500">
                  No jobs discovered yet. Click "Scan now".
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  )
}
