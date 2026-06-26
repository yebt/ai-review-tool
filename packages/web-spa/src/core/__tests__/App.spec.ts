import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import App from '../App.vue'

describe('App', () => {
  it('renders the application shell with a router outlet', () => {
    const wrapper = mount(App, {
      global: {
        stubs: {
          RouterView: { template: '<main data-test="router-view" />' },
        },
      },
    })

    expect(wrapper.find('[data-test="router-view"]').exists()).toBe(true)
  })
})
