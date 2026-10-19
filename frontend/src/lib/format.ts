// Display helpers. Decimals are formatted as strings so no value ever passes
// through a binary float. Times are stored in UTC and shown in IST.

// Groups digits the Indian way: 12,34,567.
function groupIndian(whole: string): string {
  if (whole.length <= 3) return whole
  const last3 = whole.slice(-3)
  const rest = whole.slice(0, -3).replace(/\B(?=(\d{2})+(?!\d))/g, ',')
  return rest + ',' + last3
}

// Formats a decimal string with grouping and at most `maxDecimals` places (trailing zeros dropped).
export function formatDecimal(value: string | null | undefined, maxDecimals = 4): string {
  if (value === null || value === undefined || value === '') return '—'
  const negative = value.startsWith('-')
  const [whole, frac = ''] = value.replace('-', '').split('.')
  const trimmed = frac.slice(0, maxDecimals).replace(/0+$/, '')
  return (negative ? '-' : '') + groupIndian(whole) + (trimmed ? '.' + trimmed : '')
}

// Money in rupees, always two decimals, rounded half away from zero.
export function formatMoney(value: string | null | undefined): string {
  if (value === null || value === undefined || value === '') return '—'
  const negative = value.startsWith('-')
  const [whole, frac = ''] = value.replace('-', '').split('.')
  let paise = BigInt(whole + (frac + '00').slice(0, 2))
  if (Number(frac[2] ?? '0') >= 5) paise += 1n
  const rupees = (paise / 100n).toString()
  const cents = (paise % 100n).toString().padStart(2, '0')
  return (negative && paise > 0n ? '-' : '') + '₹' + groupIndian(rupees) + '.' + cents
}

export function formatQuantity(value: string | null | undefined, unit?: string): string {
  const text = formatDecimal(value)
  return unit && text !== '—' ? `${text} ${unit}` : text
}

const istDateTime = new Intl.DateTimeFormat('en-IN', {
  timeZone: 'Asia/Kolkata',
  day: '2-digit',
  month: 'short',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  hour12: true,
})

export function formatDateTime(iso: string | null | undefined): string {
  return iso ? istDateTime.format(new Date(iso)) + ' IST' : '—'
}
