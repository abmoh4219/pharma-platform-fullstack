import { render, screen } from '@testing-library/vue'

import MatchScoreBadge from '@/components/common/MatchScoreBadge.vue'

describe('MatchScoreBadge', () => {
  test('shows score label', () => {
    render(MatchScoreBadge, {
      props: { score: 88 },
    })

    expect(screen.getByText(/match 88\/100/i)).toBeInTheDocument()
  })

  test('clamps score into range', () => {
    render(MatchScoreBadge, {
      props: { score: 999 },
    })

    expect(screen.getByText(/100\/100/i)).toBeInTheDocument()
  })
})
