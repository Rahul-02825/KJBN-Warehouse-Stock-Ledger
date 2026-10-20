const SCALE = 4

function toScaled(value: string): bigint {
  const negative = value.trim().startsWith('-')
  const [whole, fraction = ''] = value.trim().replace('-', '').split('.')
  const scaled = BigInt((whole || '0') + fraction.padEnd(SCALE, '0').slice(0, SCALE))
  return negative ? -scaled : scaled
}

function fromScaled(value: bigint): string {
  const negative = value < 0n
  const digits = (negative ? -value : value).toString().padStart(SCALE + 1, '0')
  const whole = digits.slice(0, -SCALE)
  const fraction = digits.slice(-SCALE).replace(/0+$/, '')
  return (negative ? '-' : '') + whole + (fraction ? `.${fraction}` : '')
}

export function subtractDecimal(a: string, b: string): string {
  return fromScaled(toScaled(a) - toScaled(b))
}

export function isZeroDecimal(value: string): boolean {
  return toScaled(value) === 0n
}
