import { z } from 'zod'

import type { CreateReviewRequest, ReviewComment, ReviewRecord } from './types'

const reviewCommentSchema = z.object({
  id: z.string(),
  review_id: z.string(),
  dimension: z.string(),
  severity: z.string(),
  file: z.string(),
  line_start: z.number().optional(),
  line_end: z.number().optional(),
  evidence: z.string(),
  why: z.string(),
  suggestion_snippet: z.string().optional(),
  status: z.string(),
  created_at: z.string().optional(),
})

const reviewSchema = z.object({
  id: z.string(),
  repo_id: z.string().optional(),
  project_path: z.string().optional(),
  project_url: z.string().optional(),
  platform: z.string().optional(),
  mr_id: z.string().optional(),
  mr_url: z.string().optional(),
  mr_title: z.string().optional(),
  base_sha: z.string().optional(),
  start_sha: z.string().optional(),
  head_sha: z.string().optional(),
  status: z.string(),
  scores: z.record(z.string(), z.number()).optional(),
  verdict: z.string().optional(),
  model_used: z.string().optional(),
  error: z.unknown().optional(),
  created_at: z.string().optional(),
  completed_at: z.string().optional(),
  comments: z.array(reviewCommentSchema).optional(),
})

const createReviewResponseSchema = z.object({
  review: reviewSchema,
  error: z.string().optional(),
})

const listReviewsResponseSchema = z.object({
  reviews: z.array(reviewSchema).optional().default([]),
})

const reviewResponseSchema = z.object({
  review: reviewSchema,
})

const commentsResponseSchema = z.object({
  comments: z.array(reviewCommentSchema).optional().default([]),
})

async function readJson(path: string, init: RequestInit = {}) {
  const response = await fetch(path, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...init.headers,
    },
  })

  if (!response.ok) {
    const message = await readErrorMessage(response)
    throw new Error(`${response.status} ${message}`.trim())
  }

  const text = await response.text()
  const data = text ? JSON.parse(text) : null

  return data
}

async function readErrorMessage(response: Response) {
  const fallback = response.statusText || 'Request failed'

  try {
    const text = await response.text()
    if (!text) {
      return fallback
    }

    try {
      const data: unknown = JSON.parse(text)
      if (typeof data === 'object' && data !== null) {
        const record = data as Record<string, unknown>
        const error = record.error

        if (typeof error === 'object' && error !== null && typeof (error as Record<string, unknown>).message === 'string') {
          return (error as Record<string, string>).message
        }

        if (typeof error === 'string') {
          return error
        }

        if (typeof record.message === 'string') {
          return record.message
        }
      }
    } catch {
      return fallback
    }
  } catch {
    return fallback
  }

  return fallback
}

function parseWithSchema<TData>(schema: z.ZodType<TData>, data: unknown, endpoint: string) {
  const result = schema.safeParse(data)

  if (!result.success) {
    throw new Error(`${endpoint} returned an unexpected response shape.`)
  }

  return result.data
}

export async function createReview(request: CreateReviewRequest): Promise<ReviewRecord> {
  const data = await readJson('/api/v1/reviews', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  })

  return parseWithSchema(createReviewResponseSchema, data, 'POST /api/v1/reviews').review
}

export async function listReviews(): Promise<ReviewRecord[]> {
  const data = await readJson('/api/v1/reviews')
  return parseWithSchema(listReviewsResponseSchema, data, 'GET /api/v1/reviews').reviews
}

export async function getReview(id: string): Promise<ReviewRecord> {
  const data = await readJson(`/api/v1/reviews/${encodeURIComponent(id)}`)
  return parseWithSchema(reviewResponseSchema, data, 'GET /api/v1/reviews/:id').review
}

export async function getReviewComments(id: string): Promise<ReviewComment[]> {
  const data = await readJson(`/api/v1/reviews/${encodeURIComponent(id)}/comments`)
  return parseWithSchema(commentsResponseSchema, data, 'GET /api/v1/reviews/:id/comments').comments
}
