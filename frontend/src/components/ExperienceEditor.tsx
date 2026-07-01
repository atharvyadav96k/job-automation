import { useState } from 'react'
import { Box, Button, Card, CardContent, IconButton, TextField } from '@mui/material'
import DeleteIcon from '@mui/icons-material/Delete'
import type { ExperienceItem } from '../api/profile'
import { addItem, deleteItem } from '../api/profile'

const empty = { company: '', role: '', start_date: '', end_date: '', bulletsText: '' }

interface Props {
  experience: ExperienceItem[]
  onChange: (experience: ExperienceItem[]) => void
}

export default function ExperienceEditor({ experience, onChange }: Props) {
  const [form, setForm] = useState(empty)
  const [adding, setAdding] = useState(false)

  async function handleAdd() {
    if (!form.company.trim()) return
    setAdding(true)
    try {
      const bullets = form.bulletsText.split('\n').map((b) => b.trim()).filter(Boolean)
      const created = await addItem<ExperienceItem>('experience', {
        company: form.company,
        role: form.role,
        start_date: form.start_date,
        end_date: form.end_date,
        bullets,
      })
      onChange([...experience, created])
      setForm(empty)
    } finally {
      setAdding(false)
    }
  }

  async function handleDelete(id: string) {
    await deleteItem('experience', id)
    onChange(experience.filter((e) => e.id !== id))
  }

  return (
    <Box className="flex flex-col gap-4 max-w-2xl">
      {experience.map((e) => (
        <Card key={e.id} variant="outlined">
          <CardContent className="flex justify-between items-start gap-4">
            <Box>
              <Box className="font-semibold">
                {e.role} — {e.company}
              </Box>
              <Box className="text-sm text-gray-500 mb-2">
                {e.start_date} — {e.end_date}
              </Box>
              <ul className="text-sm text-gray-700 list-disc pl-5">
                {(e.bullets ?? []).map((b, i) => (
                  <li key={i}>{b}</li>
                ))}
              </ul>
            </Box>
            <IconButton onClick={() => handleDelete(e.id)} size="small">
              <DeleteIcon fontSize="small" />
            </IconButton>
          </CardContent>
        </Card>
      ))}

      <Card variant="outlined">
        <CardContent className="flex flex-col gap-3">
          <Box className="flex flex-row gap-4">
            <TextField
              label="Company"
              value={form.company}
              onChange={(e) => setForm({ ...form, company: e.target.value })}
              size="small"
              fullWidth
            />
            <TextField
              label="Role"
              value={form.role}
              onChange={(e) => setForm({ ...form, role: e.target.value })}
              size="small"
              fullWidth
            />
          </Box>
          <Box className="flex flex-row gap-4">
            <TextField
              label="Start"
              value={form.start_date}
              onChange={(e) => setForm({ ...form, start_date: e.target.value })}
              size="small"
            />
            <TextField
              label="End"
              value={form.end_date}
              onChange={(e) => setForm({ ...form, end_date: e.target.value })}
              size="small"
            />
          </Box>
          <TextField
            label="Bullet points (one per line)"
            value={form.bulletsText}
            onChange={(e) => setForm({ ...form, bulletsText: e.target.value })}
            multiline
            minRows={3}
          />
          <Button variant="contained" onClick={handleAdd} disabled={adding} className="self-start">
            Add experience
          </Button>
        </CardContent>
      </Card>
    </Box>
  )
}
