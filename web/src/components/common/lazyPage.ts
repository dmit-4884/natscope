import { createElement, lazy, useState, type ComponentType } from 'react'

export function lazyPage<P extends object>(load: () => Promise<{ default: ComponentType<P> }>) {
  let loaded: ComponentType<P> | undefined
  const preload = () =>
    load().then((module) => {
      loaded = module.default
      return module
    })
  const Lazy = lazy(preload) as unknown as ComponentType<P>

  function Page(props: P) {
    const [{ Component }] = useState(() => ({ Component: loaded ?? Lazy }))
    return createElement(Component, props)
  }

  return Object.assign(Page, { preload })
}
