import { useState } from 'react'
import { Box, Button, Card, CardContent, IconButton, TextField } from '@mui/material'
import DeleteIcon from '@mui/icons-material/Delete'
import type { EducationItem } from '../api/profile'
import { addItem, deleteItem } from '../api/profile'

const empty = { school: '', degree: '', start_date: '', end_date: '' }

interface Props {
  education: EducationItem[]
  onChange: (education: EducationItem[]) => void
}

export default function EducationEditor({ education, onChange }: Props) {
  const [form, setForm] = useState(empty)
  const [adding, setAdding] = useState(false)

  async function handleAdd() {
    if (!form.school.trim()) return
    setAdding(true)
    try {
      const created = await addItem<EducationItem>('education', form)
      onChange([...education, created])
      setForm(empty)
    } finally {
      setAdding(false)
    }
  }

  async function handleDelete(id: string) {
    await deleteItem('education', id)
    onChange(education.filter((e) => e.id !== id))
  }

  return (
    <Box className="flex flex-col gap-4 max-w-2xl">
      {education.map((e) => (
        <Card key={e.id} variant="outlined">
          <CardContent className="flex justify-between items-start">
            <Box>
              <Box className="font-semibold">{e.school}</Box>
              <Box className="text-sm text-gray-600">{e.degree}</Box>
              <Box className="text-sm text-gray-500">
                {e.start_date} — {e.end_date}
              </Box>
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
              label="School"
              value={form.school}
              onChange={(e) => setForm({ ...form, school: e.target.value })}
              size="small"
              fullWidth
            />
            <TextField
              label="Degree"
              value={form.degree}
              onChange={(e) => setForm({ ...form, degree: e.target.value })}
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
          <Button variant="contained" onClick={handleAdd} disabled={adding} className="self-start">
            Add education
          </Button>
        </CardContent>
      </Card>
    </Box>
  )
}
