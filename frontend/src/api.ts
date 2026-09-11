export type Account = { id: string; name: string }
export const currencies = ['SGD', 'USD', 'VND'] as const
export type Currency = (typeof currencies)[number]
export type CashBalance = { currency: Currency; amount: string }

export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request(path: string, options?: RequestInit): Promise<Response> {
  let response: Response
  try {
    response = await fetch(path, { ...options, signal: AbortSignal.timeout(10_000) })
  } catch {
    throw new Error('Cannot reach the server. Check that the backend is running and try reloading.')
  }
  if (!response.ok) {
    const message = (await response.text()).trim()
    throw new ApiError(response.status, message || `Request failed (HTTP ${response.status}).`)
  }
  return response
}

async function json<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await request(path, options)
  // A missing proxy can return the frontend HTML with status 200.
  if (!response.headers.get('content-type')?.includes('application/json')) {
    throw new Error('Expected an API response. Check the API proxy configuration.')
  }
  return response.json() as Promise<T>
}

const accountPath = (id: string) => `/api/v1/accounts/${encodeURIComponent(id)}`

export const api = {
  listAccounts: () => json<{ accounts: Account[] }>('/api/v1/accounts'),
  createAccount: (name: string) => json<Account>('/api/v1/accounts', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  }),
  getAccount: (id: string) => json<Account>(accountPath(id)),
  listCash: (id: string) => json<{ balances: CashBalance[] }>(`${accountPath(id)}/cash`),
  setCash: (id: string, currency: Currency, amount: string) =>
    json<CashBalance>(`${accountPath(id)}/cash/${currency}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ amount }),
    }),
  health: async () => {
    await Promise.all(['/livez', '/readyz'].map(async (path) => {
      const response = await request(path)
      if (await response.text() !== 'OK') throw new Error(`Unexpected response from ${path}.`)
    }))
    return true
  },
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'An unexpected error occurred.'
}
