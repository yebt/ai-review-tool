export type AsyncStatus = 'idle' | 'loading' | 'success' | 'error'

export type EventConnectionStatus = 'idle' | 'connecting' | 'open' | 'closed' | 'error'

export interface AsyncState<TData> {
  status: AsyncStatus
  data: TData | null
  error: string | null
}

export interface CreateReviewRequest {
  platform: 'gitlab'
  project_url: string
  project_path: string
  mr_iid: number
}

export interface ReviewRecord {
  id: string
  repo_id?: string
  project_path?: string
  project_url?: string
  platform?: string
  mr_id?: string
  mr_url?: string
  mr_title?: string
  base_sha?: string
  start_sha?: string
  head_sha?: string
  status: string
  scores?: Record<string, number>
  verdict?: string
  model_used?: string
  error?: unknown
  created_at?: string
  completed_at?: string
  comments?: readonly ReviewComment[]
}

export interface ReviewComment {
  id: string
  review_id: string
  dimension: string
  severity: string
  file: string
  line_start?: number
  line_end?: number
  evidence: string
  why: string
  suggestion_snippet?: string
  status: string
  created_at?: string
}

export interface ReviewEvent {
  id: string
  name: string
  data: Record<string, unknown>
  receivedAt: string
}

export interface ReviewFormDraft {
  projectUrl: string
  projectPath: string
  mrIid: string
}
