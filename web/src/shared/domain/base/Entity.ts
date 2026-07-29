/** DDD Entity base; equality by identity (id), not properties. */
export abstract class Entity<T extends { id: string }> {
  protected readonly props: T
  private readonly _id: string

  protected constructor(props: T) {
    this.props = props
    this._id = props.id
  }

  get id(): string {
    return this._id
  }

  /** Equal iff same identity. */
  public equals(entity?: Entity<T>): boolean {
    if (entity === null || entity === undefined) {
      return false
    }

    if (!(entity instanceof Entity)) {
      return false
    }

    if (entity.constructor !== this.constructor) {
      return false
    }

    return this._id === entity._id
  }
}
