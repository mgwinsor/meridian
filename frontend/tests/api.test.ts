import { afterAll, afterEach, expect, spyOn, test } from 'bun:test'
import { api, ApiError } from '../src/api'

const fetchMock = spyOn(globalThis, 'fetch')
afterEach(() => fetchMock.mockReset())
afterAll(() => fetchMock.mockRestore())

test('sends exact amounts as JSON strings', async () => {
  fetchMock.mockResolvedValueOnce(Response.json({ currency: 'USD', amount: '92233720368547758.07' }))
  const balance = await api.setCash('account-id', 'USD', '92233720368547758.07')
  expect(balance.amount).toBe('92233720368547758.07')
  expect(fetchMock).toHaveBeenCalledWith('/api/v1/accounts/account-id/cash/USD', expect.objectContaining({
    method: 'PUT', body: '{"amount":"92233720368547758.07"}',
    headers: { 'Content-Type': 'application/json' },
  }))
})

test('preserves plain-text API errors and status without retrying writes', async () => {
  fetchMock.mockResolvedValueOnce(new Response('internal server error\n', { status: 500 }))
  const error = await api.createAccount('Savings').catch((error: unknown) => error)
  expect(error).toBeInstanceOf(ApiError)
  expect(error).toMatchObject({ status: 500, message: 'internal server error' })
  expect(fetchMock).toHaveBeenCalledTimes(1)
})

test('reports a network failure without retrying account creation', async () => {
  fetchMock.mockRejectedValueOnce(new TypeError('Failed to fetch'))
  await expect(api.createAccount('Savings')).rejects.toThrow('Cannot reach the server')
  expect(fetchMock).toHaveBeenCalledTimes(1)
})

test('detects an HTML fallback instead of treating it as API data', async () => {
  fetchMock.mockResolvedValueOnce(new Response('<html></html>', { headers: { 'Content-Type': 'text/html' } }))
  await expect(api.listAccounts()).rejects.toThrow('API proxy configuration')
})

test('checks both health endpoints and rejects unexpected bodies', async () => {
  fetchMock.mockImplementation(async () => new Response('not OK'))
  await expect(api.health()).rejects.toThrow('Unexpected response')
  expect(fetchMock).toHaveBeenCalledTimes(2)
})
