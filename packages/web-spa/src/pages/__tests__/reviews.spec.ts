import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import ReviewsPage from '../reviews.vue'

vi.mock('@modules/review-ops/ReviewOpsDashboard.vue', () => ({
  default: {
    name: 'ReviewOpsDashboard',
    template: '<section data-testid="review-ops-dashboard">Review ops dashboard</section>',
  },
}))

describe('ReviewsPage', () => {
  it('maps the /reviews route page to the review operations dashboard', () => {
    const wrapper = mount(ReviewsPage)

    expect(wrapper.find('[data-testid="review-ops-dashboard"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Review ops dashboard')
  })
})
