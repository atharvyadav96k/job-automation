import { useState } from 'react'
import { Box, Button, TextField, Snackbar } from '@mui/material'
import type { Basics } from '../api/profile'
import { updateBasics } from '../api/profile'

interface Props {
  basics: Basics
  onSaved: (basics: Basics) => void
}

export default function BasicsForm({ basics, onSaved }: Props) {
  const [form, setForm] = useState(basics)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)

  async function handleSave() {
    setSaving(true)
    try {
      await updateBasics(form)
      onSaved(form)
      setSaved(true)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Box className="flex flex-col gap-4 max-w-md">
      <TextField
        label="Full name"
        value={form.full_name}
        onChange={(e) => setForm({ ...form, full_name: e.target.value })}
      />
      <TextField label="Email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
      <TextField label="Phone" value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} />
      <TextField
        label="Location"
        value={form.location}
        onChange={(e) => setForm({ ...form, location: e.target.value })}
      />
      <Button variant="contained" onClick={handleSave} disabled={saving} className="self-start">
        {saving ? 'Saving…' : 'Save'}
      </Button>
      <Snackbar
        open={saved}
        autoHideDuration={2000}
        onClose={() => setSaved(false)}
        message="Saved"
      />
    </Box>
  )
}
