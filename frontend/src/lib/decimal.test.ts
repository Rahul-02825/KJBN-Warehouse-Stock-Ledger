import { describe, expect, it } from 'vitest'
import { isZeroDecimal, subtractDecimal } from './decimal'

describe('subtractDecimal', () => {
  it('subtracts exactly without floating point', () => {
    expect(subtractDecimal('8', '10')).toBe('-2')
    expect(subtractDecimal('5.5', '5.5')).toBe('0')
    expect(subtractDecimal('0.3', '0.1')).toBe('0.2')
    expect(subtractDecimal('180.25', '200')).toBe('-19.75')
    expect(subtractDecimal('3', '0')).toBe('3')
    expect(subtractDecimal('0.0001', '0')).toBe('0.0001')
  })
  it('recognises zero', () => {
    expect(isZeroDecimal('0.0000')).toBe(true)
    expect(isZeroDecimal('-0')).toBe(true)
    expect(isZeroDecimal('0.0001')).toBe(false)
  })
})
