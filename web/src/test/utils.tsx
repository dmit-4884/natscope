/* eslint-disable react-refresh/only-export-components --
 * test-only file; HMR fast-refresh export-shape rule doesn't apply here. */
import { useState } from 'react'
import type { ReactElement } from 'react'
import { render } from '@testing-library/react'
import type { RenderOptions } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

// Create a custom render function that includes QueryClientProvider
const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  })

interface AllProvidersProps {
  children: React.ReactNode
}

function AllProviders({ children }: AllProvidersProps) {
  const [client] = useState(createTestQueryClient)

  return (
    <QueryClientProvider client={client}>
      {children}
    </QueryClientProvider>
  )
}

function customRender(
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) {
  return render(ui, { wrapper: AllProviders, ...options })
}

// Re-export everything EXCEPT `render` (we override it below).
export {
  screen,
  fireEvent,
  waitFor,
  waitForElementToBeRemoved,
  act,
  cleanup,
  within,
  renderHook,
  getByRole,
  getByText,
  getByLabelText,
  getByTestId,
  getByPlaceholderText,
  queryByRole,
  queryByText,
  queryByTestId,
  findByRole,
  findByText,
} from '@testing-library/react'
export { customRender as render }
export { createTestQueryClient }
