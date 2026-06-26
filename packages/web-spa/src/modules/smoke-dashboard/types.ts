export type CheckStatus = 'idle' | 'loading' | 'success' | 'error'

export interface CheckState<TData> {
  status: CheckStatus
  data: TData | null
  error: string | null
}

export interface HealthResponse {
  status?: string
  [key: string]: unknown
}

export interface SkillHarness {
  timeout_seconds?: number
  max_retries?: number
  output_schema?: string
  require_evidence?: boolean
  min_findings_quality?: string
}

export interface ReviewSkill {
  name?: string
  description?: string
  dimension?: string
  model?: string
  readonly?: boolean
  background?: boolean
  harness?: SkillHarness
}

export interface SkillsResponse {
  skills?: ReviewSkill[]
}
