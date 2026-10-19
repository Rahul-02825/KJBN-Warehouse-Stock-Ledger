import { describe, expect, it } from 'vitest'
import { formatDecimal, formatMoney, formatQuantity } from './format'

describe('formatDecimal', () => {
  it('groups digits the Indian way', () => {
    expect(formatDecimal('1234567')).toBe('12,34,567')
    expect(formatDecimal('999')).toBe('999')
    expect(formatDecimal('1000')).toBe('1,000')
  })
  it('drops trailing zeros and keeps sign', () => {
    expect(formatDecimal('12.5000')).toBe('12.5')
    expect(formatDecimal('-3.25')).toBe('-3.25')
    expect(formatDecimal('10.0000')).toBe('10')
  })
  it('shows a dash for missing values', () => {
    expect(formatDecimal(null)).toBe('—')
    expect(formatDecimal('')).toBe('—')
  })
})

describe('formatMoney', () => {
  it('always shows two decimals', () => {
    expect(formatMoney('1100')).toBe('₹1,100.00')
    expect(formatMoney('162.5')).toBe('₹162.50')
  })
  it('rounds half away from zero', () => {
    expect(formatMoney('162.4950')).toBe('₹162.50')
    expect(formatMoney('162.4949')).toBe('₹162.49')
    expect(formatMoney('-0.005')).toBe('-₹0.01')
  })
  it('handles values too large for a float', () => {
    expect(formatMoney('123456789012345678.9999')).toBe('₹1,23,45,67,89,01,23,45,679.00')
  })
  it('does not show negative zero', () => {
    expect(formatMoney('-0.001')).toBe('₹0.00')
  })
})

describe('formatQuantity', () => {
  it('adds the unit', () => {
    expect(formatQuantity('2.5', 'kg')).toBe('2.5 kg')
    expect(formatQuantity(null, 'kg')).toBe('—')
  })
})
