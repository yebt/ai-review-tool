import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import HomePage from '../index.vue'

describe('HomePage', () => {
  it('renders the static capability map with separated action routes', () => {
    const wrapper = mount(HomePage, {
      global: {
        stubs: {
          RouterLink: {
            props: ['to'],
            template: '<a :href="String(to)"><slot /></a>',
          },
        },
      },
    })

    expect(wrapper.text()).toContain('Capability map')
    expect(wrapper.text()).toContain('Home is only a checkpoint map')
    expect(wrapper.text()).toContain('Backend health')
    expect(wrapper.text()).toContain('Loaded 4R skills')
    expect(wrapper.text()).toContain('Review operations')
    expect(wrapper.text()).toContain('Repo CRUD and repo memory management')
    expect(wrapper.text()).toContain('Publish and approval workflows')
    expect(wrapper.text()).toContain('Final Phase 7 review UI')
    expect(wrapper.text()).toContain('Go CLI workflow')

    const links = wrapper.findAll('a').map((link) => link.attributes('href'))
    expect(links).toEqual(['/health', '/skills', '/reviews', '/repos'])
  })
})
