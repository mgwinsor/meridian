import { trimInput } from './money'

export function validateQuantity(input: string): string | undefined {
  const value = trimInput(input)
  if (!/^[0-9]+(?:\.[0-9]+)?$/.test(value)) {
    return 'Enter a non-negative quantity without commas, signs, or exponents. Fractional units are allowed.'
  }
  const [whole, fraction = ''] = value.split('.')
  if (whole.replace(/^0+/, '').length > 20 || fraction.replace(/0+$/, '').length > 18) {
    return 'Use at most 20 integer digits and 18 fractional digits.'
  }
}

// The datetime-local control uses the browser's timezone. Round-trip its parts
// to reject invalid dates and local times skipped by daylight-saving changes.
export function observationTimestamp(input: string): string | undefined {
  if (!input) return undefined
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/.exec(input)
  if (!match) throw new Error('Enter a valid observation date and time.')
  const [, year, month, day, hour, minute, second = '0'] = match
  const date = new Date(input)
  if (!Number.isFinite(date.getTime()) || date.getFullYear() !== Number(year)
    || date.getMonth() + 1 !== Number(month) || date.getDate() !== Number(day)
    || date.getHours() !== Number(hour) || date.getMinutes() !== Number(minute)
    || date.getSeconds() !== Number(second)) {
    throw new Error('Enter a valid observation date and time in your local timezone.')
  }
  const timestamp = date.toISOString()
  if (!/^\d{4}-/.test(timestamp) || timestamp === '0001-01-01T00:00:00.000Z') {
    throw new Error('This observation date is outside the supported range.')
  }
  return timestamp
}
