import { lazy } from 'react'
import { describe, expect, it } from 'vitest'
import { render, screen } from '@/test/utils'
import { LazyRoute } from './LazyRoute'

const Never = lazy(() => new Promise<{ default: () => null }>(() => {}))

describe('LazyRoute', () => {
  it('holds its loading notice back so a quick chunk does not flash it', () => {
    render(
      <LazyRoute>
        <Never />
      </LazyRoute>,
    )

    expect(screen.getByText('Loading...').closest('.reveal-after-delay')).not.toBeNull()
  })
})
