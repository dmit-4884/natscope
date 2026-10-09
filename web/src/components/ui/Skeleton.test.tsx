import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { SkeletonRows } from './Skeleton'

describe('SkeletonRows', () => {
  it('stays out of sight for a quick answer', () => {
    render(<SkeletonRows count={2} />)
    expect(screen.getByRole('status', { name: 'Loading' })).toHaveClass('reveal-after-delay')
  })
})
