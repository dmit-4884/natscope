import { describe, it, expect } from 'vitest'
import { readConsumerPauseState } from './streams'

describe('readConsumerPauseState', () => {
  it('reports not paused without raw consumer info', () => {
    expect(readConsumerPauseState(undefined)).toEqual({ paused: false, pauseUntil: undefined })
  })

  it('reads paused plus the pause deadline from the server payload', () => {
    expect(
      readConsumerPauseState({
        paused: true,
        config: { pause_until: '2026-07-31T10:00:00Z' },
      }),
    ).toEqual({ paused: true, pauseUntil: '2026-07-31T10:00:00Z' })
  })

  it('treats a missing or non-boolean paused flag as not paused', () => {
    expect(readConsumerPauseState({ config: {} }).paused).toBe(false)
    expect(readConsumerPauseState({ paused: 'yes' }).paused).toBe(false)
  })

  it('ignores an empty or malformed pause_until', () => {
    expect(readConsumerPauseState({ paused: true, config: { pause_until: '' } }).pauseUntil)
      .toBeUndefined()
    expect(readConsumerPauseState({ paused: true, config: null }).pauseUntil).toBeUndefined()
    expect(readConsumerPauseState({ paused: true }).pauseUntil).toBeUndefined()
  })
})
