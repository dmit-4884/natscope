import { create } from '@bufbuild/protobuf'
import { DurationSchema } from '@bufbuild/protobuf/wkt'
import type { Duration, Timestamp } from '@bufbuild/protobuf/wkt'

/** Timestamp -> epoch millis; 0 when unset. */
export function tsToMillis(ts?: Timestamp): number {
  if (!ts) return 0
  return Number(ts.seconds) * 1000 + Math.floor(ts.nanos / 1_000_000)
}

/** Duration -> nanoseconds (UI durations are NATS-native ns); 0 when unset. */
export function durToNanos(d?: Duration): number {
  if (!d) return 0
  return Number(d.seconds) * 1_000_000_000 + d.nanos
}

/** Nanoseconds -> Duration (write side); mirrors durToNanos. */
export function nanosToDur(ns: number): Duration {
  return create(DurationSchema, {
    seconds: BigInt(Math.trunc(ns / 1_000_000_000)),
    nanos: Math.trunc(ns % 1_000_000_000),
  })
}

/** Duration -> milliseconds, or undefined when unset (optional ms knobs). */
export function durToMillis(d?: Duration): number | undefined {
  if (!d) return undefined
  return durToNanos(d) / 1_000_000
}

/** milliseconds -> Duration, or undefined when unset (optional ms knobs). */
export function millisToDur(ms?: number): Duration | undefined {
  if (ms === undefined || ms === null) return undefined
  return nanosToDur(ms * 1_000_000)
}
