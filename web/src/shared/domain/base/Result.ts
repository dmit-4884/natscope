/** Result<T,E>: Ok(value) or Err(error) for throw-free error handling. */
export class Result<T, E = Error> {
  private readonly _value?: T
  private readonly _error?: E
  private readonly _isOk: boolean

  private constructor(isOk: boolean, value?: T, error?: E) {
    this._isOk = isOk
    this._value = value
    this._error = error
  }

  static ok<T, E = Error>(value: T): Result<T, E> {
    return new Result<T, E>(true, value, undefined)
  }

  static err<T, E = Error>(error: E): Result<T, E> {
    return new Result<T, E>(false, undefined, error)
  }

  isOk(): this is Result<T, never> & { value: T } {
    return this._isOk
  }

  isErr(): this is Result<never, E> & { error: E } {
    return !this._isOk
  }

  get value(): T {
    if (!this._isOk) {
      throw new Error('Cannot get value from an error Result')
    }
    return this._value as T
  }

  get error(): E {
    if (this._isOk) {
      throw new Error('Cannot get error from a successful Result')
    }
    return this._error as E
  }

  unwrapOr(defaultValue: T): T {
    return this._isOk ? (this._value as T) : defaultValue
  }

  unwrapOrElse(fn: (error: E) => T): T {
    return this._isOk ? (this._value as T) : fn(this._error as E)
  }

  map<U>(fn: (value: T) => U): Result<U, E> {
    if (this._isOk) {
      return Result.ok(fn(this._value as T))
    }
    return Result.err(this._error as E)
  }

  mapErr<F>(fn: (error: E) => F): Result<T, F> {
    if (!this._isOk) {
      return Result.err(fn(this._error as E))
    }
    return Result.ok(this._value as T)
  }

  flatMap<U>(fn: (value: T) => Result<U, E>): Result<U, E> {
    if (this._isOk) {
      return fn(this._value as T)
    }
    return Result.err(this._error as E)
  }

  tap(fn: (value: T) => void): Result<T, E> {
    if (this._isOk) {
      fn(this._value as T)
    }
    return this
  }

  tapErr(fn: (error: E) => void): Result<T, E> {
    if (!this._isOk) {
      fn(this._error as E)
    }
    return this
  }

  fold<U>(onOk: (value: T) => U, onErr: (error: E) => U): U {
    return this._isOk ? onOk(this._value as T) : onErr(this._error as E)
  }
}

/** Combine Results into one array Result; first error short-circuits. */
export function combineResults<T, E>(results: Result<T, E>[]): Result<T[], E> {
  const values: T[] = []

  for (const result of results) {
    if (result.isErr()) {
      return Result.err(result.error)
    }
    values.push(result.value)
  }

  return Result.ok(values)
}
