import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import HealthPage from '../health.vue'

vi.mock('@modules/smoke-dashboard/HealthCheckPanel.vue', () => ({
  default: {
    name: 'HealthCheckPanel',
    template: '<section data-testid="health-check-panel">Health check panel</section>',
  },
}))

describe('HealthPage', () => {
  it('maps the /health route page to the health check panel', () => {
    const wrapper = mount(HealthPage)

    expect(wrapper.find('[data-testid="health-check-panel"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Health check panel')
  })
})
