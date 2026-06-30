import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import HealthCheckPanel from '../HealthCheckPanel.vue'

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    statusText: init.statusText,
    headers: {
      'Content-Type': 'application/json',
    },
  })
}

describe('HealthCheckPanel', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('loads only the health endpoint', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({ status: 'ok' }))

    const wrapper = mount(HealthCheckPanel)
    await flushPromises()

    expect(fetch).toHaveBeenCalledTimes(1)
    expect(fetch).toHaveBeenCalledWith('/healthz', expect.any(Object))
    expect(fetch).not.toHaveBeenCalledWith('/api/v1/skills', expect.any(Object))
    expect(wrapper.text()).toContain('Backend health')
    expect(wrapper.text()).toContain('ok')
  })
})
