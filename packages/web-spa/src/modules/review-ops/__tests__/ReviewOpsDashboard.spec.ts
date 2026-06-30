import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import ReviewOpsDashboard from '../ReviewOpsDashboard.vue'

type Listener = (event: MessageEvent<string>) => void

class MockEventSource {
  static instances: MockEventSource[] = []

  url: string
  onopen: (() => void) | null = null
  onerror: (() => void) | null = null
  closed = false
  listeners = new Map<string, Listener[]>()

  constructor(url: string) {
    this.url = url
    MockEventSource.instances.push(this)
  }

  addEventListener(name: string, listener: Listener) {
    this.listeners.set(name, [...(this.listeners.get(name) ?? []), listener])
  }

  close() {
    this.closed = true
  }

  emit(name: string, data: unknown) {
    for (const listener of this.listeners.get(name) ?? []) {
      listener(new MessageEvent(name, { data: JSON.stringify(data) }))
    }
  }
}

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    statusText: init.statusText,
    headers: {
      'Content-Type': 'application/json',
    },
  })
}

function deferredResponse() {
  let resolve!: (response: Response) => void
  const promise = new Promise<Response>((res) => {
    resolve = res
  })

  return { promise, resolve }
}

const review = {
  id: 'review_123',
  project_path: 'acme/widget',
  project_url: 'https://gitlab.com/acme/widget',
  platform: 'gitlab',
  mr_id: '7',
  mr_url: 'https://gitlab.com/acme/widget/-/merge_requests/7',
  mr_title: 'Improve widgets',
  status: 'awaiting_approval',
  scores: { risk: 80 },
  verdict: 'needs_changes',
  model_used: 'deterministic-review',
  created_at: '2026-06-30T00:00:00Z',
}

const laterReview = {
  ...review,
  id: 'review_456',
  mr_id: '8',
  mr_title: 'Current widgets',
}

const comment = {
  id: 'comment_123',
  review_id: 'review_123',
  dimension: 'risk',
  severity: 'medium',
  file: 'README.md',
  line_start: 1,
  line_end: 1,
  evidence: 'A risky line changed.',
  why: 'The change needs a safer explanation.',
  suggestion_snippet: 'Add a guardrail.',
  status: 'pending',
  created_at: '2026-06-30T00:00:00Z',
}

const laterComment = {
  ...comment,
  id: 'comment_456',
  review_id: 'review_456',
  evidence: 'Current selection comment.',
}

describe('ReviewOpsDashboard', () => {
  beforeEach(() => {
    MockEventSource.instances = []
    vi.stubGlobal('fetch', vi.fn())
    vi.stubGlobal('EventSource', MockEventSource)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('loads review history and opens detail with comments and SSE', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ reviews: [review] }))
      .mockResolvedValueOnce(jsonResponse({ review }))
      .mockResolvedValueOnce(jsonResponse({ comments: [comment] }))
      .mockResolvedValueOnce(jsonResponse({ review }))
      .mockResolvedValueOnce(jsonResponse({ comments: [comment] }))
      .mockResolvedValueOnce(jsonResponse({ reviews: [review] }))

    const wrapper = mount(ReviewOpsDashboard)
    await flushPromises()

    expect(fetch).toHaveBeenCalledWith('/api/v1/reviews', expect.any(Object))
    expect(wrapper.text()).toContain('Improve widgets')

    await wrapper.get('[data-testid="review-history-item"]').trigger('click')
    await flushPromises()

    expect(fetch).toHaveBeenCalledWith('/api/v1/reviews/review_123', expect.any(Object))
    expect(fetch).toHaveBeenCalledWith('/api/v1/reviews/review_123/comments', expect.any(Object))
    expect(MockEventSource.instances[0]?.url).toBe('/api/v1/reviews/review_123/events')
    expect(wrapper.text()).toContain('A risky line changed.')

    MockEventSource.instances[0]?.emit('agent.completed', {
      review_id: 'review_123',
      dimension: 'risk',
      attempts: 1,
    })
    MockEventSource.instances[0]?.emit('review.generated', {
      review_id: 'review_123',
      status: 'awaiting_approval',
      comments: 1,
      verdict: 'needs_changes',
    })
    await flushPromises()

    expect(wrapper.text()).toContain('agent.completed')
    expect(wrapper.text()).toContain('review.generated')
  })

  it('keeps the current selection when older detail requests resolve last', async () => {
    const firstReview = deferredResponse()
    const firstComments = deferredResponse()
    const secondReview = deferredResponse()
    const secondComments = deferredResponse()

    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ reviews: [review, laterReview] }))
      .mockReturnValueOnce(firstReview.promise)
      .mockReturnValueOnce(firstComments.promise)
      .mockReturnValueOnce(secondReview.promise)
      .mockReturnValueOnce(secondComments.promise)

    const wrapper = mount(ReviewOpsDashboard)
    await flushPromises()

    const historyItems = wrapper.findAll('[data-testid="review-history-item"]')
    expect(historyItems).toHaveLength(2)
    await historyItems[0]!.trigger('click')
    await historyItems[1]!.trigger('click')

    secondReview.resolve(jsonResponse({ review: laterReview }))
    secondComments.resolve(jsonResponse({ comments: [laterComment] }))
    await flushPromises()

    firstReview.resolve(jsonResponse({ review }))
    firstComments.resolve(jsonResponse({ comments: [comment] }))
    await flushPromises()

    expect(wrapper.text()).toContain('Current widgets')
    expect(wrapper.text()).toContain('Current selection comment.')
    expect(wrapper.text()).not.toContain('A risky line changed.')
  })

  it('creates a review with the backend request shape', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ reviews: [] }))
      .mockResolvedValueOnce(jsonResponse({ review }))
      .mockResolvedValueOnce(jsonResponse({ reviews: [review] }))
      .mockResolvedValueOnce(jsonResponse({ review }))
      .mockResolvedValueOnce(jsonResponse({ comments: [comment] }))

    const wrapper = mount(ReviewOpsDashboard)
    await flushPromises()

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(fetch).toHaveBeenCalledWith('/api/v1/reviews', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({
        platform: 'gitlab',
        project_url: 'https://gitlab.com/acme/widget',
        project_path: '',
        mr_iid: 7,
      }),
    }))
    expect(wrapper.text()).toContain('Created review review_123')
  })

  it('shows validation errors for unexpected review response shapes', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({ reviews: [{ id: 123 }] }))

    const wrapper = mount(ReviewOpsDashboard)
    await flushPromises()

    expect(wrapper.text()).toContain('GET /api/v1/reviews returned an unexpected response shape.')
  })
})
