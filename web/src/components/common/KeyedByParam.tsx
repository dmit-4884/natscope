import { Fragment, type ReactNode } from 'react'
import { useParams } from 'react-router-dom'

export function KeyedByParam({ param, children }: { param: string; children: ReactNode }) {
  const params = useParams()
  return <Fragment key={params[param]}>{children}</Fragment>
}
