import { describe, expect, test } from 'bun:test'
import { trimInput, validateAmount } from '../src/money'
import type { Currency } from '../src/api'

describe('exact money validation', () => {
  test.each<[Currency, string]>([
    ['USD', '0'], ['SGD', ' 0012.3 '], ['VND', '1000000'],
    ['USD', '92233720368547758.07'], ['SGD', '92233720368547758.07'],
    ['VND', '9223372036854775807'], ['USD', '\u008512.30\u0085'],
    ['VND', '000000000000000000000000000001'],
  ])('accepts %s %s', (currency, value) => {
    expect(validateAmount(value, currency)).toBeUndefined()
  })

  test.each<[Currency, string]>([
    ['USD', '92233720368547758.08'], ['SGD', '92233720368547758.08'],
    ['VND', '9223372036854775808'], ['USD', '1.001'], ['VND', '1.0'],
    ['USD', '-1'], ['USD', '+1'], ['SGD', '1e3'], ['VND', '1,000'],
    ['USD', '.5'], ['USD', '1.'], ['USD', ''], ['USD', '   '],
    ['USD', 'NaN'], ['USD', '\uFEFF1'],
  ])('rejects %s %s', (currency, value) => {
    expect(validateAmount(value, currency)).toBeString()
  })

  test('trims names using the backend whitespace rules', () => {
    expect(trimInput('\u0085 HSBC Premier \u3000')).toBe('HSBC Premier')
    expect(trimInput(' \t\n\u0085')).toBe('')
    expect(trimInput('\uFEFFname')).toBe('\uFEFFname')
  })
})
