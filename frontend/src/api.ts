export type Account = { id: string; name: string }
export const currencies = ['SGD', 'USD', 'VND'] as const
export type Currency = (typeof currencies)[number]
export type CashBalance = { currency: Currency; amount: string }
export type PropertyValue = { currency: Currency; amount: string }
export type Property = { id: string; name: string; value: PropertyValue }
export const instrumentKinds = ['stock', 'etf', 'bond', 'mutual_fund', 'crypto'] as const
export type InstrumentKind = (typeof instrumentKinds)[number]
export const instrumentKindLabels: Record<InstrumentKind, string> = {
  stock: 'Stock',
  etf: 'ETF',
  bond: 'Bond',
  mutual_fund: 'Mutual fund',
  crypto: 'Crypto',
}
export const investmentClasses = ['equities', 'bonds', 'crypto', 'mixed', 'unclassified'] as const
export type InvestmentClass = (typeof investmentClasses)[number]
export type AssetClass = InvestmentClass | 'cash' | 'property'
export type InstrumentInput = {
  kind: InstrumentKind
  symbol: string
  name: string
  quoteCurrency: Currency
  assetClass?: InvestmentClass
}
export type Instrument = InstrumentInput & { id: string; assetClass: InvestmentClass }
export type Position = { accountId: string; instrumentId: string; quantity: string }
export type PriceObservation = {
  instrumentId: string
  currency: Currency
  amount: string
  observedAt: string
}

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
  const data = (await response.json()) as T
  if (options?.method === 'POST' || options?.method === 'PUT') {
    writeRevision++
    for (const listener of writeListeners) listener()
  }
  return data
}

const accountPath = (id: string) => `/api/v1/accounts/${encodeURIComponent(id)}`
const instrumentPath = (id: string) => `/api/v1/instruments/${encodeURIComponent(id)}`

const writeListeners = new Set<() => void>()
let writeRevision = 0
export const valuationChanges = {
  subscribe(listener: () => void) {
    writeListeners.add(listener)
    return () => {
      writeListeners.delete(listener)
    }
  },
  getSnapshot: () => writeRevision,
}

export type FXObservation = { currency: 'SGD' | 'VND'; rate: string; observedAt: string }
export type Breakdown = { key: string; amount: string; percentage?: string; excludedAssets: number }
export type OverviewAsset = {
  source: 'cash' | 'position' | 'property'
  id: string
  accountId?: string
  instrumentId?: string
  name: string
  assetClass: AssetClass
  currency: Currency
  quantity?: string
  nativeValue?: string
  convertedValue?: string
  price?: Omit<PriceObservation, 'instrumentId'>
  missingInputs: string[]
}
export type OverviewReport = {
  currency: Currency
  generatedAt: string
  complete: boolean
  valuedTotal: string
  assets: OverviewAsset[]
  assetClasses: Breakdown[]
  currencies: Breakdown[]
  fxObservations: FXObservation[]
}

export const api = {
  overview: (currency: Currency) => json<OverviewReport>(`/api/v1/overview?currency=${currency}`),
  setAssetClass: (id: string, assetClass: InvestmentClass) =>
    json<Instrument>(`${instrumentPath(id)}/asset-class`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ assetClass }),
    }),
  listFX: (currency: 'SGD' | 'VND') =>
    json<{ observations: FXObservation[] }>(`/api/v1/fx-rates/${currency}`),
  recordFX: (currency: 'SGD' | 'VND', rate: string, observedAt?: string) =>
    json<FXObservation>(`/api/v1/fx-rates/${currency}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ rate, observedAt }),
    }),
  listInstruments: () => json<{ instruments: Instrument[] }>('/api/v1/instruments'),
  createInstrument: (instrument: InstrumentInput) =>
    json<Instrument>('/api/v1/instruments', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(instrument),
    }),
  getInstrument: (id: string) => json<Instrument>(instrumentPath(id)),
  listPositions: (accountId: string) =>
    json<{ positions: Position[] }>(`${accountPath(accountId)}/positions`),
  setPosition: (accountId: string, instrumentId: string, quantity: string) =>
    json<Position>(`${accountPath(accountId)}/positions/${encodeURIComponent(instrumentId)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ quantity }),
    }),
  listPrices: (instrumentId: string) =>
    json<{ observations: PriceObservation[] }>(`${instrumentPath(instrumentId)}/prices`),
  recordPrice: (instrumentId: string, amount: string, observedAt?: string) =>
    json<PriceObservation>(`${instrumentPath(instrumentId)}/prices`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ amount, observedAt }),
    }),
  listAccounts: () => json<{ accounts: Account[] }>('/api/v1/accounts'),
  createAccount: (name: string) =>
    json<Account>('/api/v1/accounts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    }),
  getAccount: (id: string) => json<Account>(accountPath(id)),
  listProperties: () => json<{ properties: Property[] }>('/api/v1/properties'),
  createProperty: (name: string, value: PropertyValue) =>
    json<Property>('/api/v1/properties', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, value }),
    }),
  setPropertyValue: (propertyId: string, value: PropertyValue) =>
    json<Property>(`/api/v1/properties/${encodeURIComponent(propertyId)}/value`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(value),
    }),
  listCash: (id: string) => json<{ balances: CashBalance[] }>(`${accountPath(id)}/cash`),
  setCash: (id: string, currency: Currency, amount: string) =>
    json<CashBalance>(`${accountPath(id)}/cash/${currency}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ amount }),
    }),
  health: async () => {
    await Promise.all(
      ['/livez', '/readyz'].map(async (path) => {
        const response = await request(path)
        if ((await response.text()) !== 'OK') throw new Error(`Unexpected response from ${path}.`)
      })
    )
    return true
  },
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'An unexpected error occurred.'
}
