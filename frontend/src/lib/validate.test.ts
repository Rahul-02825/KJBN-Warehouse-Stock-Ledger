import { describe, expect, it } from 'vitest'
import { checkDecimal, checkText, onlyErrors } from './validate'

describe('checkDecimal', () => {
  it('accepts positive numbers with up to 4 decimals', () => {
    expect(checkDecimal('12.5', 'kg')).toBe('')
    expect(checkDecimal('0.0001', 'kg')).toBe('')
  })
  it('rejects bad input', () => {
    expect(checkDecimal('', 'kg')).toBe('Required')
    expect(checkDecimal('-1', 'kg')).toBe('Must not be negative')
    expect(checkDecimal('1.23456', 'kg')).toMatch(/4 decimal/)
    expect(checkDecimal('abc', 'kg')).toMatch(/number/)
    expect(checkDecimal('0', 'kg')).toBe('Must be greater than zero')
  })
  it('requires whole numbers for pcs', () => {
    expect(checkDecimal('2.5', 'pcs')).toMatch(/whole number/)
    expect(checkDecimal('2.0', 'pcs')).toBe('')
  })
  it('can allow zero or be optional', () => {
    expect(checkDecimal('0', 'pcs', { allowZero: true })).toBe('')
    expect(checkDecimal('', 'pcs', { required: false })).toBe('')
  })
})

describe('checkText', () => {
  it('checks required and length', () => {
    expect(checkText(' ', 'SKU', 64)).toBe('SKU is required')
    expect(checkText('x'.repeat(65), 'SKU', 64)).toMatch(/64/)
    expect(checkText('', 'Note', 10, false)).toBe('')
  })
})

describe('onlyErrors', () => {
  it('drops empty messages', () => {
    expect(onlyErrors({ a: '', b: 'bad' })).toEqual({ b: 'bad' })
  })
})
