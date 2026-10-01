import { expect, afterEach, vi } from 'vitest'
import { cleanup } from '@testing-library/react'
import * as matchers from '@testing-library/jest-dom/matchers'

// Extend Vitest's expect with jest-dom matchers
expect.extend(matchers)

// Mock clipboard API for jsdom
Object.defineProperty(navigator, 'clipboard', {
  value: {
    writeText: vi.fn().mockResolvedValue(undefined),
    readText: vi.fn().mockResolvedValue(''),
  },
  writable: true,
  configurable: true,
})

function readBlob<T extends string | ArrayBuffer>(blob: Blob, as: 'text' | 'buffer'): Promise<T> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as T)
    reader.onerror = () => reject(reader.error)
    if (as === 'text') reader.readAsText(blob)
    else reader.readAsArrayBuffer(blob)
  })
}

Blob.prototype.text ??= function text(this: Blob) {
  return readBlob<string>(this, 'text')
}
Blob.prototype.arrayBuffer ??= function arrayBuffer(this: Blob) {
  return readBlob<ArrayBuffer>(this, 'buffer')
}

// Cleanup after each test
afterEach(() => {
  cleanup()
})
