import { fireEvent, render, screen } from '@testing-library/vue'

import LoginForm from '@/components/common/LoginForm.vue'

describe('LoginForm', () => {
  test('emits submit payload with credentials', async () => {
    const { emitted } = render(LoginForm)

    const username = screen.getByLabelText(/username/i)
    const password = screen.getByLabelText(/password/i)

    await fireEvent.update(username, 'admin')
    await fireEvent.update(password, 'Admin123!')
    await fireEvent.click(screen.getByTestId('login-submit'))

    const submitEvents = emitted().submit
    expect(submitEvents).toBeTruthy()
    expect(submitEvents[0][0]).toEqual({
      username: 'admin',
      password: 'Admin123!',
    })
  })
})
