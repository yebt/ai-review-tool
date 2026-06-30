import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import SkillsPage from '../skills.vue'

vi.mock('@modules/smoke-dashboard/SkillsPanel.vue', () => ({
  default: {
    name: 'SkillsPanel',
    template: '<section data-testid="skills-panel">Skills panel</section>',
  },
}))

describe('SkillsPage', () => {
  it('maps the /skills route page to the skills panel', () => {
    const wrapper = mount(SkillsPage)

    expect(wrapper.find('[data-testid="skills-panel"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Skills panel')
  })
})
