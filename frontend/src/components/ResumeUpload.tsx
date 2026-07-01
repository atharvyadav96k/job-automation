import { useRef, useState } from 'react'
import { Alert, Box, Button, LinearProgress, Snackbar, Typography } from '@mui/material'
import { uploadResume } from '../api/profile'

interface Props {
  currentPath: string
  onUploaded: (path: string) => void
}

export default function ResumeUpload({ currentPath, onUploaded }: Props) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [uploaded, setUploaded] = useState(false)

  async function handleFile(file: File) {
    if (!file.name.toLowerCase().endsWith('.docx')) {
      setError('Only .docx files are accepted')
      return
    }
    setError(null)
    setUploading(true)
    setProgress(0)
    try {
      const { path } = await uploadResume(file, setProgress)
      onUploaded(path)
      setUploaded(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Upload failed')
    } finally {
      setUploading(false)
    }
  }

  return (
    <Box className="flex flex-col gap-3 max-w-md">
      <Typography variant="body2" className="text-gray-600">
        Current template: {currentPath || 'none uploaded yet'}
      </Typography>
      <input
        ref={inputRef}
        type="file"
        accept=".docx"
        hidden
        onChange={(e) => e.target.files?.[0] && handleFile(e.target.files[0])}
      />
      <Button variant="contained" onClick={() => inputRef.current?.click()} disabled={uploading} className="self-start">
        {uploading ? `Uploading… ${progress}%` : 'Upload base resume (.docx)'}
      </Button>
      {uploading && <LinearProgress variant="determinate" value={progress} />}
      {error && <Alert severity="error">{error}</Alert>}
      <Snackbar
        open={uploaded}
        autoHideDuration={2500}
        onClose={() => setUploaded(false)}
        message="Resume uploaded"
      />
    </Box>
  )
}
