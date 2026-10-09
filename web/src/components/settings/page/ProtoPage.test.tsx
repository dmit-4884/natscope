import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@/test/utils'
import ProtoPage from './ProtoPage'

vi.mock('@/contexts/proto', () => ({ useSchemaTypes: () => ({ data: undefined }) }))
vi.mock('@/components/proto/ProtoManager', () => ({ default: () => null }))

describe('ProtoPage', () => {
  it('claims no proto files only once the types have loaded', () => {
    render(<ProtoPage />)

    expect(screen.queryByText('No proto files loaded')).not.toBeInTheDocument()
  })
})
