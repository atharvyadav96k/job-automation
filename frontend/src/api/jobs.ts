import { apiFetch } from './client'

export interface JobSummary {
  id: number
  title: string
  company_name: string
  status: string
  url: string
  discovered_at: string
  version_count: number
  latest_match_score: number | null
  latest_ats_score: number | null
  approved: boolean
}

export interface JobContext {
  id: number
  company_summary: string
  key_requirements: string[]
  inferred_tone: string
}

export interface ATSBreakdown {
  matched_keywords: string[]
  missing_keywords: string[]
  formatting_notes: string[]
}

export interface TailoredContent {
  skills: Record<string, string>
  exp1_bullets: string[]
  projects: { title: string; tech: string; link: string }[]
}

export interface ResumeVersion {
  id: number
  version_number: number
  match_score: number | null
  ats_score: number | null
  ats_score_breakdown: ATSBreakdown
  tailored_content: TailoredContent
  generated_cover_letter_text: string
  changes_summary: string
  reasoning: string
  model_used: string
  approved: boolean
  is_active: boolean
  created_at: string
}

export interface JobDetail {
  id: number
  title: string
  company_name: string
  status: string
  url: string
  description_clean: string
  discovered_at: string
  job_context: JobContext | null
  resume_versions: ResumeVersion[]
}

export function listJobs() {
  return apiFetch<JobSummary[]>('/api/jobs')
}

export function getJob(id: number) {
  return apiFetch<JobDetail>(`/api/jobs/${id}`)
}

export function tailorJob(id: number) {
  return apiFetch<{ ResumeVersionID: number }>(`/api/jobs/${id}/tailor`, { method: 'POST' })
}

export function approveVersion(jobId: number, versionId: number) {
  return apiFetch<void>(`/api/jobs/${jobId}/resume-versions/${versionId}/approve`, { method: 'POST' })
}

export function rejectJob(jobId: number) {
  return apiFetch<void>(`/api/jobs/${jobId}/reject`, { method: 'POST' })
}

export function runDiscovery() {
  return apiFetch<{ sources: number; queued: number }>('/api/discovery/run', { method: 'POST' })
}
