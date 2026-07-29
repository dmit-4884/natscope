import { Entity } from './Entity'

export abstract class AggregateRoot<T extends { id: string }> extends Entity<T> {}
