import { useEffect, useState } from 'react'
import {
  Box,
  Button,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Link,
  MenuItem,
  Paper,
  TextField,
  Typography,
  Alert,
} from '@mui/material'
import type { JobDetail, ResumeVersion } from '../api/jobs'
import { getJob, approveVersion, rejectJob, markApplied, recordReply, tailorJob } from '../api/jobs'
import TextDiff from '../components/TextDiff'

interface Props {
  jobId: number
  onBack: () => void
}

export default function JobDetailPage({ jobId, onBack }: Props) {
  const [job, setJob] = useState<JobDetail | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busyVersionId, setBusyVersionId] = useState<number | null>(null)
  const [diffFromId, setDiffFromId] = useState<number | ''>('')
  const [diffToId, setDiffToId] = useState<number | ''>('')
  const [markingApplied, setMarkingApplied] = useState(false)
  const [generatingResume, setGeneratingResume] = useState(false)
  const [replyDialogOpen, setReplyDialogOpen] = useState(false)
  const [replyChannel, setReplyChannel] = useState('')
  const [replyOutcome, setReplyOutcome] = useState<'interview' | 'rejected' | 'offer' | 'other'>('interview')
  const [savingReply, setSavingReply] = useState(false)

  async function load() {
    const detail = await getJob(jobId)
    setJob(detail)
    if (detail.resume_versions.length >= 2) {
      const sorted = [...detail.resume_versions].sort((a, b) => a.version_number - b.version_number)
      setDiffFromId(sorted[sorted.length - 2].id)
      setDiffToId(sorted[sorted.length - 1].id)
    }
  }

  useEffect(() => {
    load()
  }, [jobId])

  async function handleApprove(versionId: number) {
    setBusyVersionId(versionId)
    setError(null)
    try {
      await approveVersion(jobId, versionId)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Approve failed')
    } finally {
      setBusyVersionId(null)
    }
  }

  async function handleReject() {
    setError(null)
    try {
      await rejectJob(jobId)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Reject failed')
    }
  }

  async function handleGenerateResume() {
    setGeneratingResume(true)
    setError(null)
    try {
      await tailorJob(jobId)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Resume generation failed')
    } finally {
      setGeneratingResume(false)
    }
  }

  async function handleMarkApplied() {
    setMarkingApplied(true)
    setError(null)
    try {
      await markApplied(jobId)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Mark as applied failed')
    } finally {
      setMarkingApplied(false)
    }
  }

  async function handleSaveReply() {
    if (!job?.application) return
    setSavingReply(true)
    setError(null)
    try {
      await recordReply(job.application.id, replyChannel, replyOutcome)
      setReplyDialogOpen(false)
      setReplyChannel('')
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Recording reply failed')
    } finally {
      setSavingReply(false)
    }
  }

  if (!job) {
    return (
      <Box className="flex justify-center items-center min-h-screen">
        <CircularProgress />
      </Box>
    )
  }

  const versions = [...job.resume_versions].sort((a, b) => b.version_number - a.version_number)
  const diffFrom = job.resume_versions.find((v) => v.id === diffFromId)
  const diffTo = job.resume_versions.find((v) => v.id === diffToId)

  return (
    <Box className="max-w-4xl mx-auto p-6">
      <Button onClick={onBack} className="mb-4">
        ← Back to jobs
      </Button>

      <Typography variant="h4">{job.title}</Typography>
      <Typography variant="subtitle1" className="text-gray-600 mb-1">
        {job.company_name} · {job.status}
      </Typography>
      <Link href={job.url} target="_blank" rel="noreferrer" className="text-sm">
        View original posting
      </Link>

      {error && (
        <Alert severity="error" className="my-4">
          {error}
        </Alert>
      )}

      <Box className="flex gap-2 my-4 items-center">
        <Button variant="outlined" color="error" onClick={handleReject}>
          Reject job
        </Button>

        <Button variant="contained" onClick={handleGenerateResume} disabled={generatingResume}>
          {generatingResume ? 'Generating…' : job.resume_versions.length > 0 ? 'Re-generate resume' : 'Generate resume'}
        </Button>

        {!job.application && (
          <Button
            variant="outlined"
            onClick={handleMarkApplied}
            disabled={markingApplied || !job.resume_versions.some((v) => v.approved)}
          >
            {markingApplied ? 'Marking…' : 'Mark as applied'}
          </Button>
        )}

        {job.application && !job.application.replied_at && (
          <Button variant="contained" onClick={() => setReplyDialogOpen(true)}>
            I got a reply
          </Button>
        )}

        {job.application?.replied_at && (
          <Chip
            label={`Reply via ${job.application.reply_channel} · ${job.application.outcome} · ${new Date(
              job.application.replied_at,
            ).toLocaleDateString()}`}
            color="info"
          />
        )}
      </Box>

      <Dialog open={replyDialogOpen} onClose={() => setReplyDialogOpen(false)}>
        <DialogTitle>Where did the reply come from?</DialogTitle>
        <DialogContent className="flex flex-col gap-4 pt-2 min-w-80">
          <TextField
            label="Channel (e.g. email, LinkedIn, portal, phone)"
            value={replyChannel}
            onChange={(e) => setReplyChannel(e.target.value)}
            autoFocus
          />
          <TextField
            select
            label="Outcome"
            value={replyOutcome}
            onChange={(e) => setReplyOutcome(e.target.value as typeof replyOutcome)}
          >
            <MenuItem value="interview">Interview request</MenuItem>
            <MenuItem value="rejected">Rejected</MenuItem>
            <MenuItem value="offer">Offer</MenuItem>
            <MenuItem value="other">Other</MenuItem>
          </TextField>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setReplyDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleSaveReply} disabled={savingReply || !replyChannel.trim()}>
            {savingReply ? 'Saving…' : 'Save'}
          </Button>
        </DialogActions>
      </Dialog>

      {(job.apply_url || job.contact_email) && (
        <Paper variant="outlined" className="p-4 mb-6">
          <Typography variant="h6" className="mb-2">
            How to apply
          </Typography>
          {job.apply_url && (
            <Typography variant="body2" className="mb-1">
              <Link href={job.apply_url} target="_blank" rel="noreferrer">
                Apply via {job.apply_portal ?? 'this link'}
              </Link>
            </Typography>
          )}
          {job.contact_email && (
            <Typography variant="body2">
              Contact: <Link href={`mailto:${job.contact_email}`}>{job.contact_email}</Link>
            </Typography>
          )}
        </Paper>
      )}

      <Paper variant="outlined" className="p-4 mb-6">
        <Typography variant="h6" className="mb-2">
          Job description
        </Typography>
        <Typography variant="body2" className="whitespace-pre-wrap text-gray-700">
          {job.description_clean}
        </Typography>
      </Paper>

      {job.job_context && (
        <Paper variant="outlined" className="p-4 mb-6">
          <Typography variant="h6" className="mb-2">
            Job context
          </Typography>
          <Typography variant="body2" className="mb-2">
            {job.job_context.company_summary}
          </Typography>
          <Typography variant="body2" className="text-gray-600 mb-2">
            Tone: {job.job_context.inferred_tone}
          </Typography>
          <Box className="flex flex-wrap gap-1">
            {job.job_context.key_requirements.map((req, i) => (
              <Chip key={i} size="small" label={req} />
            ))}
          </Box>
        </Paper>
      )}

      {job.resume_versions.length >= 2 && (
        <Paper variant="outlined" className="p-4 mb-6">
          <Typography variant="h6" className="mb-3">
            Compare versions
          </Typography>
          <Box className="flex gap-4 mb-4">
            <TextField
              select
              label="From"
              size="small"
              value={diffFromId}
              onChange={(e) => setDiffFromId(Number(e.target.value))}
              className="w-40"
            >
              {job.resume_versions.map((v) => (
                <MenuItem key={v.id} value={v.id}>
                  v{v.version_number}
                </MenuItem>
              ))}
            </TextField>
            <TextField
              select
              label="To"
              size="small"
              value={diffToId}
              onChange={(e) => setDiffToId(Number(e.target.value))}
              className="w-40"
            >
              {job.resume_versions.map((v) => (
                <MenuItem key={v.id} value={v.id}>
                  v{v.version_number}
                </MenuItem>
              ))}
            </TextField>
          </Box>
          {diffFrom && diffTo && (
            <>
              <Typography variant="subtitle2" className="mb-1">
                Cover letter diff
              </Typography>
              <TextDiff before={diffFrom.generated_cover_letter_text} after={diffTo.generated_cover_letter_text} />
              <Divider className="my-3" />
              <Typography variant="subtitle2" className="mb-1">
                Experience bullets diff
              </Typography>
              <TextDiff
                before={diffFrom.tailored_content.exp1_bullets?.join('\n') ?? ''}
                after={diffTo.tailored_content.exp1_bullets?.join('\n') ?? ''}
              />
            </>
          )}
        </Paper>
      )}

      <Typography variant="h6" className="mb-3">
        Resume versions
      </Typography>
      <Box className="flex flex-col gap-4">
        {versions.map((v) => (
          <VersionCard key={v.id} version={v} busy={busyVersionId === v.id} onApprove={() => handleApprove(v.id)} />
        ))}
      </Box>
    </Box>
  )
}

function VersionCard({
  version,
  busy,
  onApprove,
}: {
  version: ResumeVersion
  busy: boolean
  onApprove: () => void
}) {
  return (
    <Paper variant="outlined" className="p-4">
      <Box className="flex justify-between items-start mb-2">
        <Box>
          <Typography variant="subtitle1">
            Version {version.version_number}
            {version.is_active && <Chip size="small" label="active" color="primary" className="ml-2" />}
            {version.approved && <Chip size="small" label="approved" color="success" className="ml-1" />}
          </Typography>
          <Typography variant="body2" className="text-gray-600">
            Match: {version.match_score ?? '—'} · ATS: {version.ats_score ?? '—'} · {version.model_used}
          </Typography>
        </Box>
        <Button size="small" variant="contained" onClick={onApprove} disabled={busy || version.approved}>
          {busy ? 'Approving…' : version.approved ? 'Approved' : 'Approve'}
        </Button>
      </Box>

      <Typography variant="body2" className="mb-1">
        <strong>Changes:</strong> {version.changes_summary}
      </Typography>
      <Typography variant="body2" className="mb-3">
        <strong>Reasoning:</strong> {version.reasoning}
      </Typography>

      <Box className="mb-3">
        <Typography variant="subtitle2">Matched keywords</Typography>
        <Box className="flex flex-wrap gap-1 mb-2">
          {version.ats_score_breakdown?.matched_keywords?.map((k, i) => (
            <Chip key={i} size="small" color="success" label={k} />
          ))}
        </Box>
        <Typography variant="subtitle2">Missing keywords</Typography>
        <Box className="flex flex-wrap gap-1">
          {version.ats_score_breakdown?.missing_keywords?.map((k, i) => (
            <Chip key={i} size="small" color="warning" label={k} />
          ))}
        </Box>
      </Box>

      <Typography variant="subtitle2">Cover letter</Typography>
      <Typography variant="body2" className="whitespace-pre-wrap text-gray-700">
        {version.generated_cover_letter_text}
      </Typography>
    </Paper>
  )
}
