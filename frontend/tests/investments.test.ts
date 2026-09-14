import { expect, test } from 'bun:test'
import { observationTimestamp, validateQuantity } from '../src/investments'

test('quantities preserve arbitrary precision and magnitude independently of money limits', () => {
  for (const value of ['0', '000.000', ' 0012.3400\u0085', '0.000000000000000000001', '9223372036854775807999.1234567890123456789']) {
    expect(validateQuantity(value)).toBeUndefined()
  }
})

test('quantities reject signs, exponents, separators and malformed decimals', () => {
  for (const value of ['', ' ', '-1', '+1', '1e3', '.5', '1.', '1,000', '1.2.3', '\ufeff1', 'NaN']) {
    expect(validateQuantity(value)).toBeDefined()
  }
})

test('optional observation time is omitted and local times convert to UTC', () => {
  expect(observationTimestamp('')).toBeUndefined()
  expect(observationTimestamp('2026-09-14T12:34:56')).toBe(new Date(2026, 8, 14, 12, 34, 56).toISOString())
  expect(observationTimestamp('2024-02-29T12:34')).toBe(new Date(2024, 1, 29, 12, 34).toISOString())
})

test('observation time rejects malformed dates and rollover instead of silently changing the observation', () => {
  for (const value of ['invalid', '2026-02-30T12:00', '2025-02-29T12:00', '2026-09-14T24:00', '2026-09-14', '2026-09-14T12:00Z']) {
    expect(() => observationTimestamp(value)).toThrow('valid observation date and time')
  }
})
