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
  console.log(`API integration passed through ${origin}: ${requests.length} requests covering all 7 operations.`)
  console.log(`Created test account ${account.id}; it remains until the backend restarts.`)
} finally {
  globalThis.fetch = originalFetch
}
