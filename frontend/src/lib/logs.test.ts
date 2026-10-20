import { describe, expect, it } from 'vitest'
import { logDetails } from './logs'

describe('logDetails', () => {
  it('lists every field except the ones shown in their own columns', () => {
    const line = { time: 't', level: 'WARN', msg: 'request rejected', request_id: 'abc', path: '/api/movements/out', product_id: 1, quantity: '9' }
    expect(logDetails(line)).toEqual([
      ['path', '/api/movements/out'],
      ['product_id', '1'],
      ['quantity', '9'],
    ])
  })
})
