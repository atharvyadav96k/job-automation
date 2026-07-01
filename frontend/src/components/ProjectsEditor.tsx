import { useState } from 'react'
import { Box, Button, Card, CardContent, IconButton, Link, TextField } from '@mui/material'
import DeleteIcon from '@mui/icons-material/Delete'
import type { ProjectItem } from '../api/profile'
import { addItem, deleteItem } from '../api/profile'

const empty = { title: '', description: '', tech_stack: '', link: '', bulletsText: '' }

interface Props {
  projects: ProjectItem[]
  onChange: (projects: ProjectItem[]) => void
}

export default function ProjectsEditor({ projects, onChange }: Props) {
  const [form, setForm] = useState(empty)
  const [adding, setAdding] = useState(false)

  async function handleAdd() {
    if (!form.title.trim()) return
    setAdding(true)
    try {
      const bullets = form.bulletsText.split('\n').map((b) => b.trim()).filter(Boolean)
      const created = await addItem<ProjectItem>('projects', {
        title: form.title,
        description: form.description,
        tech_stack: form.tech_stack,
        link: form.link,
        bullets,
      })
      onChange([...projects, created])
      setForm(empty)
    } finally {
      setAdding(false)
    }
  }

  async function handleDelete(id: string) {
    await deleteItem('projects', id)
    onChange(projects.filter((p) => p.id !== id))
  }

  return (
    <Box className="flex flex-col gap-4 max-w-2xl">
      {projects.map((p) => (
        <Card key={p.id} variant="outlined">
          <CardContent className="flex justify-between items-start gap-4">
            <Box>
              <Box className="font-semibold">{p.title}</Box>
              <Box className="text-sm text-gray-500 mb-1">{p.tech_stack}</Box>
              {p.link && (
                <Link href={p.link} target="_blank" rel="noreferrer" className="text-sm">
                  {p.link}
                </Link>
              )}
              <ul className="text-sm text-gray-700 list-disc pl-5">
                {(p.bullets ?? []).map((b, i) => (
                  <li key={i}>{b}</li>
                ))}
              </ul>
            </Box>
            <IconButton onClick={() => handleDelete(p.id)} size="small">
              <DeleteIcon fontSize="small" />
            </IconButton>
          </CardContent>
        </Card>
      ))}

      <Card variant="outlined">
        <CardContent className="flex flex-col gap-3">
          <TextField
            label="Title"
            value={form.title}
            onChange={(e) => setForm({ ...form, title: e.target.value })}
            size="small"
          />
          <TextField
            label="Description"
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
            size="small"
          />
          <TextField
            label="Tech stack"
            value={form.tech_stack}
            onChange={(e) => setForm({ ...form, tech_stack: e.target.value })}
            size="small"
          />
          <TextField
            label="Link"
            value={form.link}
            onChange={(e) => setForm({ ...form, link: e.target.value })}
            size="small"
          />
          <TextField
            label="Bullet points (one per line)"
            value={form.bulletsText}
            onChange={(e) => setForm({ ...form, bulletsText: e.target.value })}
            multiline
            minRows={3}
          />
          <Button variant="contained" onClick={handleAdd} disabled={adding} className="self-start">
            Add project
          </Button>
        </CardContent>
      </Card>
    </Box>
  )
}
