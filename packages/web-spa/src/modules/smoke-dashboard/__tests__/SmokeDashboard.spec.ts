import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import SmokeDashboard from '../SmokeDashboard.vue'

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    statusText: init.statusText,
    headers: {
      'Content-Type': 'application/json',
    },
  })
}

describe('SmokeDashboard', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('renders idle states before the first endpoint responses resolve', () => {
    vi.mocked(fetch).mockImplementation(() => new Promise<Response>(() => {}))

    const wrapper = mount(SmokeDashboard)

    expect(wrapper.text()).toContain('Not loaded')
    expect(wrapper.text()).toContain('Health has not been requested yet.')
    expect(wrapper.text()).toContain('Skills have not been requested yet.')
  })

  it('loads health and skills endpoint data', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ status: 'ok' }))
      .mockResolvedValueOnce(
        jsonResponse({
          skills: [
            {
              name: 'review-readability',
              description: 'Checks readability.',
              dimension: 'readability',
              harness: {
                output_schema: 'readability-findings',
                max_retries: 2,
                timeout_seconds: 30,
              },
            },
          ],
        }),
      )

    const wrapper = mount(SmokeDashboard)
    await flushPromises()

    expect(fetch).toHaveBeenCalledWith('/healthz', expect.any(Object))
    expect(fetch).toHaveBeenCalledWith('/api/v1/skills', expect.any(Object))
    expect(wrapper.text()).toContain('ok')
    expect(wrapper.text()).toContain('review-readability')
    expect(wrapper.text()).toContain('readability-findings')
    expect(wrapper.text()).toContain('Skills loaded')
  })

  it('shows useful endpoint errors and allows refresh', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ error: { message: 'health unavailable' } }, { status: 503 }))
      .mockResolvedValueOnce(jsonResponse({ error: { message: 'skills unavailable' } }, { status: 500 }))
      .mockResolvedValueOnce(jsonResponse({ status: 'ok' }))
      .mockResolvedValueOnce(jsonResponse({ skills: [] }))

    const wrapper = mount(SmokeDashboard)
    await flushPromises()

    expect(wrapper.text()).toContain('503 health unavailable')
    expect(wrapper.text()).toContain('500 skills unavailable')

    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(fetch).toHaveBeenCalledTimes(4)
    expect(wrapper.text()).toContain('The endpoint returned no skills.')
  })

  it('shows friendly validation errors for unexpected endpoint shapes', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ status: 200 }))
      .mockResolvedValueOnce(jsonResponse({ skills: [{ name: 123 }] }))

    const wrapper = mount(SmokeDashboard)
    await flushPromises()

    expect(wrapper.text()).toContain('/healthz returned an unexpected response shape.')
    expect(wrapper.text()).toContain('/api/v1/skills returned an unexpected response shape.')
  })
})
