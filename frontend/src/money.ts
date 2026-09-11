import type { Currency } from './api'

// Match Go strings.TrimSpace, including U+0085 (which JS trim does not remove).
export function trimInput(value: string): string {
  return value.replace(/^[\t\n\v\f\r \u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]+|[\t\n\v\f\r \u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]+$/g, '')
}

export function validateAmount(input: string, currency: Currency): string | undefined {
  const amount = trimInput(input)
  const pattern = currency === 'VND' ? /^[0-9]+$/ : /^[0-9]+(?:\.[0-9]{1,2})?$/
  if (!pattern.test(amount)) {
    return currency === 'VND'
      ? 'Enter a non-negative whole number without commas for VND.'
      : 'Enter a non-negative amount with at most two decimal places, without commas.'
  }
  const [whole, fraction = ''] = amount.split('.')
  const minorUnits = (whole + (currency === 'VND' ? '' : fraction.padEnd(2, '0'))).replace(/^0+/, '') || '0'
  const maximum = '9223372036854775807'
  if (minorUnits.length > maximum.length || (minorUnits.length === maximum.length && minorUnits > maximum)) {
    return `Amount exceeds the maximum ${currency === 'VND' ? maximum : '92233720368547758.07'} ${currency}.`
  }
}
