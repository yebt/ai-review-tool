import { computed, readonly, shallowRef } from 'vue'
import { z } from 'zod'

import type { CheckState, HealthResponse, ReviewSkill } from './types'

const healthResponseSchema = z.object({
  status: z.string().optional(),
}).passthrough()

const skillHarnessSchema = z.object({
  timeout_seconds: z.number().optional(),
  max_retries: z.number().optional(),
  output_schema: z.string().optional(),
  require_evidence: z.boolean().optional(),
  min_findings_quality: z.string().optional(),
})

const reviewSkillSchema = z.object({
  name: z.string().optional(),
  description: z.string().optional(),
  dimension: z.string().optional(),
  model: z.string().optional(),
  readonly: z.boolean().optional(),
  background: z.boolean().optional(),
  harness: skillHarnessSchema.optional(),
})

const skillsResponseSchema = z.object({
  skills: z.array(reviewSkillSchema).optional().default([]),
})

function createIdleState<TData>(): CheckState<TData> {
  return {
    status: 'idle',
    data: null,
    error: null,
  }
}

function validateResponse<TData>(schema: z.ZodType<TData>, data: unknown, endpointName: string) {
  const result = schema.safeParse(data)

  if (!result.success) {
    throw new Error(`${endpointName} returned an unexpected response shape.`)
  }

  return result.data
}

async function readJson(path: string): Promise<unknown> {
  const response = await fetch(path, {
    headers: {
      Accept: 'application/json',
    },
  })

  const text = await response.text()
  const data = text ? JSON.parse(text) : null

  if (!response.ok) {
    const message = data?.error?.message ?? data?.message ?? response.statusText
    throw new Error(`${response.status} ${message}`.trim())
  }

  return data
}

export function useHealthCheck() {
  const health = shallowRef<CheckState<HealthResponse>>(createIdleState())

  async function loadHealth() {
    health.value = { status: 'loading', data: null, error: null }

    try {
      health.value = {
        status: 'success',
        data: validateResponse(healthResponseSchema, await readJson('/healthz'), '/healthz'),
        error: null,
      }
    } catch (error) {
      health.value = {
        status: 'error',
        data: null,
        error: error instanceof Error ? error.message : 'Unknown health check error',
      }
    }
  }

  return {
    health: readonly(health),
    loadHealth,
  }
}

export function useSkillsCheck() {
  const skills = shallowRef<CheckState<ReviewSkill[]>>(createIdleState())
  const skillsCount = computed(() => skills.value.data?.length ?? 0)

  async function loadSkills() {
    skills.value = { status: 'loading', data: null, error: null }

    try {
      const data = validateResponse(skillsResponseSchema, await readJson('/api/v1/skills'), '/api/v1/skills')
      skills.value = {
        status: 'success',
        data: Array.isArray(data.skills) ? data.skills : [],
        error: null,
      }
    } catch (error) {
      skills.value = {
        status: 'error',
        data: null,
        error: error instanceof Error ? error.message : 'Unknown skills check error',
      }
    }
  }

  return {
    skills: readonly(skills),
    skillsCount,
    loadSkills,
  }
}
