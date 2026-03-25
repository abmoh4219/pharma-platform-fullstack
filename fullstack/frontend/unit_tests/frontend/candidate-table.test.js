import { fireEvent, render, screen } from '@testing-library/vue'

import CandidateTable from '@/components/common/CandidateTable.vue'

describe('CandidateTable', () => {
  test('renders candidate rows and emits edit', async () => {
    const candidates = [
      {
        id: 1,
        full_name: 'John Doe',
        phone: '***-***-1234',
        id_number: '****1234',
        position_title: 'Pharmacist',
        status: 'new',
      },
    ]

    const { emitted } = render(CandidateTable, {
      props: { candidates, loading: false },
    })

    expect(screen.getByText('John Doe')).toBeInTheDocument()
    expect(screen.getAllByTestId('candidate-row')).toHaveLength(1)

    await fireEvent.click(screen.getByRole('button', { name: /edit/i }))
    expect(emitted().edit).toBeTruthy()
    expect(emitted().edit[0][0]).toEqual(candidates[0])
  })

  test('shows loading state', () => {
    render(CandidateTable, {
      props: { candidates: [], loading: true },
    })

    expect(screen.getByText(/loading candidates/i)).toBeInTheDocument()
  })
})
