import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import SkillsPanel from '../SkillsPanel.vue'

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    statusText: init.statusText,
    headers: {
      'Content-Type': 'application/json',
    },
  })
}

describe('SkillsPanel', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('loads only the skills endpoint', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({
      skills: [
        {
          name: 'review-risk',
          description: 'Checks risk.',
          dimension: 'risk',
          harness: {
            output_schema: 'risk-findings',
            max_retries: 2,
            timeout_seconds: 30,
          },
        },
      ],
    }))

    const wrapper = mount(SkillsPanel)
    await flushPromises()

    expect(fetch).toHaveBeenCalledTimes(1)
    expect(fetch).toHaveBeenCalledWith('/api/v1/skills', expect.any(Object))
    expect(fetch).not.toHaveBeenCalledWith('/healthz', expect.any(Object))
    expect(wrapper.text()).toContain('Loaded 4R skills')
    expect(wrapper.text()).toContain('review-risk')
    expect(wrapper.text()).toContain('risk-findings')
  })
})
