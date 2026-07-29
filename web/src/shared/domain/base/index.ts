// Base domain building blocks, used across every bounded context.
export { ValueObject } from './ValueObject'
export { Entity } from './Entity'
export { AggregateRoot } from './AggregateRoot'
export { Result, combineResults } from './Result'
export {
  DomainError,
  type DomainErrorKind,
  type DomainErrorInit,
} from './DomainError'
