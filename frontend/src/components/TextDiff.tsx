import { diffWords } from 'diff'
import { Box } from '@mui/material'

interface Props {
  before: string
  after: string
}

export default function TextDiff({ before, after }: Props) {
  const parts = diffWords(before ?? '', after ?? '')
  return (
    <Box className="whitespace-pre-wrap text-sm leading-relaxed">
      {parts.map((part, i) => {
        if (part.added) {
          return (
            <span key={i} className="bg-green-100 text-green-900">
              {part.value}
            </span>
          )
        }
        if (part.removed) {
          return (
            <span key={i} className="bg-red-100 text-red-900 line-through">
              {part.value}
            </span>
          )
        }
        return <span key={i}>{part.value}</span>
      })}
    </Box>
  )
}
