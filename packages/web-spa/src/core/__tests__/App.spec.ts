import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import App from '../App.vue'

describe('App', () => {
  it('renders the application shell with a router outlet', () => {
    const wrapper = mount(App, {
      global: {
        stubs: {
          RouterLink: {
            props: ['to'],
            template: '<a :href="String(to)"><slot /></a>',
          },
          RouterView: { template: '<main data-test="router-view" />' },
        },
      },
    })

    expect(wrapper.find('[data-test="router-view"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Co-Review Smoke UI')
    expect(wrapper.text()).toContain('Health')
    expect(wrapper.text()).toContain('Skills')
    expect(wrapper.text()).toContain('Reviews')
    expect(wrapper.text()).toContain('Repos')
  })
})
