import { apiFetch, API_BASE, authHeader, ApiError } from './client'

export type ArrayField = 'skills' | 'experience' | 'education' | 'projects'

export interface SkillItem {
  id: string
  name: string
  category: string
}

export interface ExperienceItem {
  id: string
  company: string
  role: string
  start_date: string
  end_date: string
  bullets: string[]
}

export interface EducationItem {
  id: string
  school: string
  degree: string
  start_date: string
  end_date: string
}

export interface ProjectItem {
  id: string
  title: string
  description: string
  tech_stack: string
  link: string
  bullets: string[]
}

export interface Profile {
  id: number
  full_name: string
  email: string
  phone: string
  location: string
  base_resume_docx_path: string
  skills: SkillItem[]
  experience: ExperienceItem[]
  education: EducationItem[]
  projects: ProjectItem[]
}

export interface Basics {
  full_name: string
  email: string
  phone: string
  location: string
}

export function getProfile() {
  return apiFetch<Profile>('/api/profile')
}

export function updateBasics(basics: Basics) {
  return apiFetch<void>('/api/profile', { method: 'PUT', body: JSON.stringify(basics) })
}

export function addItem<T>(field: ArrayField, item: Omit<T, 'id'>) {
  return apiFetch<T>(`/api/profile/${field}`, { method: 'POST', body: JSON.stringify(item) })
}

export function updateItem<T>(field: ArrayField, id: string, item: Omit<T, 'id'>) {
  return apiFetch<void>(`/api/profile/${field}/${id}`, { method: 'PUT', body: JSON.stringify(item) })
}

export function deleteItem(field: ArrayField, id: string) {
  return apiFetch<void>(`/api/profile/${field}/${id}`, { method: 'DELETE' })
}

// XMLHttpRequest is used instead of fetch here because fetch has no
// upload-progress event; onProgress drives the UI's progress bar.
export function uploadResume(file: File, onProgress?: (percent: number) => void) {
  const form = new FormData()
  form.append('resume', file)

  return new Promise<{ path: string }>((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('POST', `${API_BASE}/api/profile/resume`)
    const auth = authHeader()
    if (auth) xhr.setRequestHeader('Authorization', auth)

    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) onProgress(Math.round((e.loaded / e.total) * 100))
    }
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(JSON.parse(xhr.responseText))
      } else {
        reject(new ApiError(xhr.status, xhr.responseText || xhr.statusText))
      }
    }
    xhr.onerror = () => reject(new ApiError(0, 'network error'))
    xhr.send(form)
  })
}
