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

test('creates, lists, and updates properties with exact values and encoded identities', async () => {
  const property = { id: 'property/id', name: 'Home', value: { currency: 'VND' as const, amount: '9223372036854775807' } }
  fetchMock.mockResolvedValueOnce(Response.json(property))
  expect(await api.createProperty('Home', property.value)).toEqual(property)
  expect(fetchMock).toHaveBeenLastCalledWith('/api/v1/properties', expect.objectContaining({
    method: 'POST', body: JSON.stringify({ name: 'Home', value: property.value }),
    headers: { 'Content-Type': 'application/json' },
  }))
  fetchMock.mockResolvedValueOnce(Response.json({ properties: [property] }))
  expect(await api.listProperties()).toEqual({ properties: [property] })
  expect(fetchMock).toHaveBeenLastCalledWith('/api/v1/properties', expect.anything())
  const value = { currency: 'USD' as const, amount: '92233720368547758.07' }
  fetchMock.mockResolvedValueOnce(Response.json({ ...property, value }))
  expect(await api.setPropertyValue(property.id, value)).toEqual({ ...property, value })
  expect(fetchMock).toHaveBeenLastCalledWith('/api/v1/properties/property%2Fid/value', expect.objectContaining({
    method: 'PUT', body: JSON.stringify(value), headers: { 'Content-Type': 'application/json' },
  }))
})

test('does not retry uncertain property creation or value replacement', async () => {
  const value = { currency: 'USD' as const, amount: '1' }
  fetchMock.mockRejectedValueOnce(new TypeError('Failed to fetch'))
  await expect(api.createProperty('Home', value)).rejects.toThrow('Cannot reach the server')
  expect(fetchMock).toHaveBeenCalledTimes(1)
  fetchMock.mockResolvedValueOnce(new Response('internal server error\n', { status: 500 }))
  await expect(api.setPropertyValue('property', value)).rejects.toMatchObject({ status: 500, message: 'internal server error' })
  expect(fetchMock).toHaveBeenCalledTimes(2)
})

test('checks both health endpoints and rejects unexpected bodies', async () => {
  fetchMock.mockImplementation(async () => new Response('not OK'))
  await expect(api.health()).rejects.toThrow('Unexpected response')
  expect(fetchMock).toHaveBeenCalledTimes(2)
})
