import assert from 'node:assert/strict'
import { api, ApiError } from '../src/api'

// Exercise the same client used by React, through Vite's same-origin proxy.
const origin = process.env.API_TEST_ORIGIN || 'http://127.0.0.1:5173'
const originalFetch = globalThis.fetch
const requests: string[] = []
globalThis.fetch = ((input: string | URL | Request, init?: RequestInit) => {
  requests.push(`${init?.method ?? 'GET'} ${input}`)
  return originalFetch(new URL(String(input), origin), init)
}) as typeof fetch

try {
  await api.health()
  const account = await api.createAccount(`  Frontend smoke ${Date.now()}  `)
  assert.equal(account.name, account.name.trim())
  assert.deepEqual(await api.getAccount(account.id), account)
  assert.ok((await api.listAccounts()).accounts.some((item) => item.id === account.id))
  assert.deepEqual((await api.listCash(account.id)).balances, [])

  for (const [currency, amount, canonical] of [
    ['USD', '0012.3', '12.30'],
    ['SGD', '2500', '2500.00'],
    ['VND', '9223372036854775807', '9223372036854775807'],
    ['USD', '92233720368547758.07', '92233720368547758.07'],
    ['SGD', '0', '0.00'],
  ] as const) {
    assert.deepEqual(await api.setCash(account.id, currency, amount), { currency, amount: canonical })
  }
  assert.deepEqual((await api.listCash(account.id)).balances, [
    { currency: 'SGD', amount: '0.00' },
    { currency: 'USD', amount: '92233720368547758.07' },
    { currency: 'VND', amount: '9223372036854775807' },
  ])
  await assert.rejects(api.createAccount('  '), (error: unknown) =>
    error instanceof ApiError && error.status === 400 && error.message === 'account name cannot be empty')
  await assert.rejects(api.setCash(account.id, 'VND', '1.5'), (error: unknown) =>
    error instanceof ApiError && error.status === 400 && error.message === 'invalid amount')
  await assert.rejects(api.setCash(account.id, 'USD', '92233720368547758.08'), (error: unknown) =>
    error instanceof ApiError && error.status === 400 && error.message === 'invalid amount')
  await assert.rejects(api.getAccount('00000000-0000-0000-0000-000000000000'), (error: unknown) =>
    error instanceof ApiError && error.status === 404 && error.message === 'account not found')
  await api.listProperties()
  const property = await api.createProperty('  Home  ', { currency: 'SGD', amount: '0750000.5' })
  assert.equal(property.name, 'Home')
  assert.deepEqual(property.value, { currency: 'SGD', amount: '750000.50' })
  const duplicate = await api.createProperty('Home', property.value)
  assert.notEqual(property.id, duplicate.id)
  for (const [currency, amount, canonical] of [
    ['VND', '9223372036854775807', '9223372036854775807'],
    ['USD', '92233720368547758.07', '92233720368547758.07'],
    ['SGD', '0', '0.00'],
  ] as const) {
    assert.deepEqual(await api.setPropertyValue(property.id, { currency, amount }), {
      ...property, value: { currency, amount: canonical },
    })
  }
  await assert.rejects(api.createProperty(' ', { currency: 'USD', amount: '1' }), (error: unknown) =>
    error instanceof ApiError && error.status === 400 && error.message === 'property name cannot be empty')
  await assert.rejects(api.setPropertyValue(property.id, { currency: 'USD', amount: '92233720368547758.08' }), (error: unknown) =>
    error instanceof ApiError && error.status === 400 && error.message === 'invalid amount')
  await assert.rejects(api.setPropertyValue('00000000-0000-0000-0000-000000000000', { currency: 'USD', amount: '1' }), (error: unknown) =>
    error instanceof ApiError && error.status === 404 && error.message === 'property not found')
  const properties = (await api.listProperties()).properties.filter((item) => item.id === property.id || item.id === duplicate.id)
  assert.deepEqual(properties, [duplicate, { ...property, value: { currency: 'SGD', amount: '0.00' } }]
    .sort((a, b) => a.id.localeCompare(b.id)))
  const instruments = []
  for (const kind of ['stock', 'etf', 'bond', 'mutual_fund', 'crypto'] as const) {
    const instrument = await api.createInstrument({ name: `  Smoke ${kind}  `, symbol: ' TEST ', kind, quoteCurrency: 'USD' })
    assert.equal(instrument.name, `Smoke ${kind}`)
    assert.equal(instrument.symbol, 'TEST')
    assert.deepEqual(await api.getInstrument(instrument.id), instrument)
    instruments.push(instrument)
  }
  const instrument = instruments[0]
  assert.ok((await api.listInstruments()).instruments.some((item) => item.id === instrument.id))
  assert.deepEqual((await api.listPositions(account.id)).positions, [])
  for (const [quantity, canonical] of [
    ['0012.3400', '12.34'], ['9223372036854775807999.000000000000000001', '9223372036854775807999.000000000000000001'], ['0.000', '0'],
  ]) {
    const expected = { accountId: account.id, instrumentId: instrument.id, quantity: canonical }
    assert.deepEqual(await api.setPosition(account.id, instrument.id, quantity), expected)
    assert.deepEqual((await api.listPositions(account.id)).positions, [expected])
  }
  assert.deepEqual((await api.listPrices(instrument.id)).observations, [])
  const observedAt = '2026-09-14T12:00:00.123456789+08:00'
  const price = await api.recordPrice(instrument.id, '92233720368547758.07', observedAt)
  assert.deepEqual(price, { instrumentId: instrument.id, currency: 'USD', amount: '92233720368547758.07', observedAt: '2026-09-14T04:00:00.123456789Z' })
  assert.deepEqual(await api.recordPrice(instrument.id, price.amount, observedAt), price)
  const earlier = await api.recordPrice(instrument.id, '0', '2020-01-01T00:00:00Z')
  assert.deepEqual((await api.listPrices(instrument.id)).observations, [earlier, price, price])
  const now = await api.recordPrice(instrument.id, '0012.3')
  assert.equal(now.amount, '12.30')
  assert.ok(Number.isFinite(Date.parse(now.observedAt)))
  const vnd = await api.createInstrument({ name: 'VND stock', symbol: 'VND', kind: 'stock', quoteCurrency: 'VND' })
  assert.equal((await api.recordPrice(vnd.id, '9223372036854775807')).amount, '9223372036854775807')
  const invalid = (error: unknown) => error instanceof ApiError && error.status === 400
  const missing = (error: unknown) => error instanceof ApiError && error.status === 404
  await assert.rejects(api.createInstrument({ name: ' ', symbol: 'X', kind: 'stock', quoteCurrency: 'USD' }), invalid)
  await assert.rejects(api.setPosition(account.id, instrument.id, '1e3'), invalid)
  await assert.rejects(api.recordPrice(vnd.id, '1.5'), invalid)
  await assert.rejects(api.recordPrice(instrument.id, '92233720368547758.08'), invalid)
  await assert.rejects(api.recordPrice(instrument.id, '1', ''), invalid)
  const missingId = '00000000-0000-0000-0000-000000000000'
  await assert.rejects(api.getInstrument(missingId), missing)
  await assert.rejects(api.listPositions(missingId), missing)
  await assert.rejects(api.setPosition(account.id, missingId, '1'), missing)
  await assert.rejects(api.listPrices(missingId), missing)
  console.log(`API integration passed through ${origin}: ${requests.length} requests covering all 17 operations.`)
  console.log(`Created test account ${account.id}; it remains until the backend restarts.`)
} finally {
  globalThis.fetch = originalFetch
}
