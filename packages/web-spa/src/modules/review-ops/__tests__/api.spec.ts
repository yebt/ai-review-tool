import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { listReviews } from '../api'

function response(body: string, init: ResponseInit) {
  return new Response(body, init)
}

describe('review ops api', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it.each([
    ['empty body', '', 'Bad Request'],
    ['malformed JSON body', '{', 'Bad Request'],
    ['HTML body', '<h1>bad gateway</h1>', 'Bad Request'],
    ['JSON message shape', '{"message":"review failed"}', 'review failed'],
    ['JSON nested error shape', '{"error":{"message":"missing token"}}', 'missing token'],
    ['JSON string error shape', '{"error":"not allowed"}', 'not allowed'],
  ])('reports HTTP errors for %s', async (_caseName, body, expectedMessage) => {
    vi.mocked(fetch).mockResolvedValueOnce(response(body, { status: 400, statusText: 'Bad Request' }))

    await expect(listReviews()).rejects.toThrow(`400 ${expectedMessage}`)
  })
})
