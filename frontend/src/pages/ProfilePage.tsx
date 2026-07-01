import { useEffect, useState } from 'react'
import { Box, CircularProgress, Tab, Tabs, Typography } from '@mui/material'
import type { Profile } from '../api/profile'
import { getProfile } from '../api/profile'
import BasicsForm from '../components/BasicsForm'
import SkillsEditor from '../components/SkillsEditor'
import ExperienceEditor from '../components/ExperienceEditor'
import EducationEditor from '../components/EducationEditor'
import ProjectsEditor from '../components/ProjectsEditor'
import ResumeUpload from '../components/ResumeUpload'

const TABS = ['Basics', 'Skills', 'Experience', 'Education', 'Projects', 'Resume'] as const

export default function ProfilePage() {
  const [profile, setProfile] = useState<Profile | null>(null)
  const [tab, setTab] = useState(0)

  useEffect(() => {
    getProfile().then(setProfile)
  }, [])

  if (!profile) {
    return (
      <Box className="flex justify-center items-center min-h-screen">
        <CircularProgress />
      </Box>
    )
  }

  return (
    <Box className="max-w-3xl mx-auto p-6">
      <Typography variant="h4" className="mb-6">
        My Profile
      </Typography>
      <Tabs value={tab} onChange={(_, v) => setTab(v)} className="mb-6">
        {TABS.map((label) => (
          <Tab key={label} label={label} />
        ))}
      </Tabs>

      {tab === 0 && <BasicsForm basics={profile} onSaved={(b) => setProfile({ ...profile, ...b })} />}
      {tab === 1 && (
        <SkillsEditor skills={profile.skills} onChange={(skills) => setProfile({ ...profile, skills })} />
      )}
      {tab === 2 && (
        <ExperienceEditor
          experience={profile.experience}
          onChange={(experience) => setProfile({ ...profile, experience })}
        />
      )}
      {tab === 3 && (
        <EducationEditor
          education={profile.education}
          onChange={(education) => setProfile({ ...profile, education })}
        />
      )}
      {tab === 4 && (
        <ProjectsEditor projects={profile.projects} onChange={(projects) => setProfile({ ...profile, projects })} />
      )}
      {tab === 5 && (
        <ResumeUpload
          currentPath={profile.base_resume_docx_path}
          onUploaded={(path) => setProfile({ ...profile, base_resume_docx_path: path })}
        />
      )}
    </Box>
  )
}
