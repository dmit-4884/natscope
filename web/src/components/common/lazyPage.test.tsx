import { Suspense } from 'react'
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { lazyPage } from './lazyPage'

function Page({ name }: { name: string }) {
  return <p>page {name}</p>
}

describe('lazyPage', () => {
  it('renders a preloaded page at once, without the loading fallback', async () => {
    const Route = lazyPage(async () => ({ default: Page }))
    await Route.preload()

    render(
      <Suspense fallback={<p>loading</p>}>
        <Route name="config" />
      </Suspense>,
    )

    expect(screen.getByText('page config')).toBeInTheDocument()
    expect(screen.queryByText('loading')).not.toBeInTheDocument()
  })

  it('still loads a page that was not preloaded', async () => {
    const Route = lazyPage(async () => ({ default: Page }))

    render(
      <Suspense fallback={<p>loading</p>}>
        <Route name="relations" />
      </Suspense>,
    )

    expect(await screen.findByText('page relations')).toBeInTheDocument()
  })
})
