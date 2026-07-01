import { useState } from 'react'
import { Box, Button, Chip, MenuItem, TextField } from '@mui/material'
import DeleteIcon from '@mui/icons-material/Delete'
import type { SkillItem } from '../api/profile'
import { addItem, deleteItem } from '../api/profile'

const CATEGORIES = ['backend', 'frontend', 'database', 'integration', 'devops', 'cloud', 'tools']

interface Props {
  skills: SkillItem[]
  onChange: (skills: SkillItem[]) => void
}

export default function SkillsEditor({ skills, onChange }: Props) {
  const [name, setName] = useState('')
  const [category, setCategory] = useState(CATEGORIES[0])
  const [adding, setAdding] = useState(false)

  async function handleAdd() {
    if (!name.trim()) return
    setAdding(true)
    try {
      const created = await addItem<SkillItem>('skills', { name: name.trim(), category })
      onChange([...skills, created])
      setName('')
    } finally {
      setAdding(false)
    }
  }

  async function handleDelete(id: string) {
    await deleteItem('skills', id)
    onChange(skills.filter((s) => s.id !== id))
  }

  const byCategory = CATEGORIES.map((cat) => ({
    cat,
    items: skills.filter((s) => s.category === cat),
  })).filter((g) => g.items.length > 0)

  return (
    <Box className="flex flex-col gap-6 max-w-2xl">
      <Box className="flex flex-row gap-4 items-center">
        <TextField label="Skill name" value={name} onChange={(e) => setName(e.target.value)} size="small" />
        <TextField
          select
          label="Category"
          value={category}
          onChange={(e) => setCategory(e.target.value)}
          size="small"
          className="w-40"
        >
          {CATEGORIES.map((c) => (
            <MenuItem key={c} value={c}>
              {c}
            </MenuItem>
          ))}
        </TextField>
        <Button variant="contained" onClick={handleAdd} disabled={adding}>
          Add
        </Button>
      </Box>

      {byCategory.map(({ cat, items }) => (
        <Box key={cat}>
          <Box className="text-sm font-semibold uppercase text-gray-500 mb-2">{cat}</Box>
          <Box className="flex flex-row flex-wrap gap-2">
            {items.map((skill) => (
              <Chip
                key={skill.id}
                label={skill.name}
                onDelete={() => handleDelete(skill.id)}
                deleteIcon={<DeleteIcon />}
              />
            ))}
          </Box>
        </Box>
      ))}
      {skills.length === 0 && <Box className="text-gray-500">No skills yet.</Box>}
    </Box>
  )
}
