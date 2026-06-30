import { computed, onUnmounted, readonly, shallowRef } from 'vue'

import { createReview, getReview, getReviewComments, listReviews } from './api'
import type { AsyncState, CreateReviewRequest, EventConnectionStatus, ReviewComment, ReviewEvent, ReviewRecord } from './types'

type EventSourceFactory = (url: string) => EventSource

function idleState<TData>(): AsyncState<TData> {
  return {
    status: 'idle',
    data: null,
    error: null,
  }
}

function errorMessage(error: unknown, fallback: string) {
  return error instanceof Error ? error.message : fallback
}

function parseEventData(data: string): Record<string, unknown> {
  if (!data) {
    return {}
  }

  try {
    const parsed = JSON.parse(data)
    return typeof parsed === 'object' && parsed !== null ? parsed as Record<string, unknown> : { value: parsed }
  } catch {
    return { raw: data }
  }
}

function createBrowserEventSource(url: string) {
  return new EventSource(url)
}

export function useReviewOps(eventSourceFactory: EventSourceFactory = createBrowserEventSource) {
  const history = shallowRef<AsyncState<ReviewRecord[]>>(idleState())
  const selectedReview = shallowRef<AsyncState<ReviewRecord>>(idleState())
  const comments = shallowRef<AsyncState<ReviewComment[]>>(idleState())
  const createState = shallowRef<AsyncState<ReviewRecord>>(idleState())
  const selectedReviewId = shallowRef<string | null>(null)
  const events = shallowRef<ReviewEvent[]>([])
  const eventConnectionStatus = shallowRef<EventConnectionStatus>('idle')
  const eventSource = shallowRef<EventSource | null>(null)

  const selectedCommentsCount = computed(() => comments.value.data?.length ?? 0)

  async function refreshHistory() {
    history.value = { status: 'loading', data: history.value.data, error: null }

    try {
      history.value = {
        status: 'success',
        data: await listReviews(),
        error: null,
      }
    } catch (error) {
      history.value = {
        status: 'error',
        data: null,
        error: errorMessage(error, 'Could not load review history'),
      }
    }
  }

  async function loadReview(id: string) {
    const requestId = id
    selectedReviewId.value = id
    selectedReview.value = { status: 'loading', data: selectedReview.value.data, error: null }
    comments.value = { status: 'loading', data: comments.value.data, error: null }
    connectEvents(id)

    try {
      const [review, reviewComments] = await Promise.all([getReview(id), getReviewComments(id)])

      if (selectedReviewId.value !== requestId) {
        return
      }

      selectedReview.value = { status: 'success', data: review, error: null }
      comments.value = { status: 'success', data: reviewComments, error: null }
    } catch (error) {
      if (selectedReviewId.value !== requestId) {
        return
      }

      selectedReview.value = {
        status: 'error',
        data: null,
        error: errorMessage(error, 'Could not load review detail'),
      }
      comments.value = {
        status: 'error',
        data: null,
        error: errorMessage(error, 'Could not load review comments'),
      }
    }
  }

  async function submitReview(request: CreateReviewRequest) {
    createState.value = { status: 'loading', data: null, error: null }

    try {
      const review = await createReview(request)
      createState.value = { status: 'success', data: review, error: null }
      await refreshHistory()
      await loadReview(review.id)
    } catch (error) {
      createState.value = {
        status: 'error',
        data: null,
        error: errorMessage(error, 'Could not create review'),
      }
    }
  }

  function connectEvents(reviewId: string) {
    closeEvents()
    events.value = []
    eventConnectionStatus.value = 'connecting'
    const source = eventSourceFactory(`/api/v1/reviews/${encodeURIComponent(reviewId)}/events`)
    eventSource.value = source

    const eventNames = ['review.started', 'agent.started', 'agent.completed', 'agent.error', 'review.generated', 'review.error']
    for (const name of eventNames) {
      source.addEventListener(name, (event) => {
        const message = event as MessageEvent<string>
        events.value = [
          ...events.value,
          {
            id: `${name}-${events.value.length}-${Date.now()}`,
            name,
            data: parseEventData(message.data),
            receivedAt: new Date().toISOString(),
          },
        ]

        if (name === 'review.generated' || name === 'review.error') {
          void loadSelectedSnapshot(reviewId)
        }
      })
    }

    source.onopen = () => {
      eventConnectionStatus.value = 'open'
    }
    source.onerror = () => {
      eventConnectionStatus.value = 'error'
    }
  }

  async function loadSelectedSnapshot(reviewId: string) {
    if (selectedReviewId.value !== reviewId) {
      return
    }

    try {
      const [review, reviewComments] = await Promise.all([getReview(reviewId), getReviewComments(reviewId)])

      if (selectedReviewId.value !== reviewId) {
        return
      }

      selectedReview.value = { status: 'success', data: review, error: null }
      comments.value = { status: 'success', data: reviewComments, error: null }
      await refreshHistory()
    } catch {
      // Keep the visible event log. Detail refresh errors are already recoverable by selecting the review again.
    }
  }

  function closeEvents() {
    eventSource.value?.close()
    eventSource.value = null
    eventConnectionStatus.value = selectedReviewId.value ? 'closed' : 'idle'
  }

  onUnmounted(() => {
    closeEvents()
  })

  return {
    history: readonly(history),
    selectedReview: readonly(selectedReview),
    comments: readonly(comments),
    createState: readonly(createState),
    selectedReviewId: readonly(selectedReviewId),
    events: readonly(events),
    eventConnectionStatus: readonly(eventConnectionStatus),
    selectedCommentsCount,
    refreshHistory,
    loadReview,
    submitReview,
    closeEvents,
  }
}
