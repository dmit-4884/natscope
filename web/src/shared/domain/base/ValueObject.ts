/** DDD Value Object base; immutable, equality by properties not identity. */
export abstract class ValueObject<T extends object> {
  protected readonly props: Readonly<T>

  protected constructor(props: T) {
    this.props = Object.freeze({ ...props })
  }

  /** Equal iff all properties equal. */
  public equals(vo?: ValueObject<T>): boolean {
    if (vo === null || vo === undefined) {
      return false
    }

    if (vo.constructor !== this.constructor) {
      return false
    }

    return this.shallowEquals(
      this.props as Record<string, unknown>,
      vo.props as Record<string, unknown>
    )
  }

  /** Shallow equality, recursing into nested ValueObjects. */
  private shallowEquals(obj1: Record<string, unknown>, obj2: Record<string, unknown>): boolean {
    const keys1 = Object.keys(obj1)
    const keys2 = Object.keys(obj2)

    if (keys1.length !== keys2.length) {
      return false
    }

    return keys1.every((key) => {
      const val1 = obj1[key]
      const val2 = obj2[key]

      if (val1 instanceof ValueObject && val2 instanceof ValueObject) {
        return val1.equals(val2)
      }

      return val1 === val2
    })
  }

  /** New instance with patched props (immutable update). */
  protected clone(props: Partial<T>): this {
    const Constructor = this.constructor as new (props: T) => this
    return new Constructor({ ...this.props, ...props })
  }
}
