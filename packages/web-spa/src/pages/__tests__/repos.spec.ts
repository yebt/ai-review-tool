import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import ReposPage from '../repos.vue'

describe('ReposPage', () => {
  it('clearly marks repo CRUD as unavailable until Phase 5', () => {
    const wrapper = mount(ReposPage)

    expect(wrapper.text()).toContain('Repo CRUD not available')
    expect(wrapper.text()).toContain('Phase 5')
    expect(wrapper.text()).toContain('Placeholder only')
  })
})
