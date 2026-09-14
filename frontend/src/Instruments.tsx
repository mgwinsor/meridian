import { useCallback, useState } from 'react'
import type { SubmitEvent } from 'react'
import { api, ApiError, currencies, errorMessage, instrumentKinds, instrumentKindLabels } from './api'
import type { Currency, Instrument, InstrumentKind } from './api'
import { trimInput } from './money'
import { useResource } from './useResource'
import { Prices } from './Prices'

export type InstrumentCatalog = ReturnType<typeof useResource<{ instruments: Instrument[] }>>

function CreateInstrument({ onCreated }: { onCreated: (instrument: Instrument) => void }) {
  const [name, setName] = useState('')
  const [symbol, setSymbol] = useState('')
  const [kind, setKind] = useState<InstrumentKind>('stock')
  const [quoteCurrency, setQuoteCurrency] = useState<Currency>('USD')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault()
    if (saving) return
    if (!trimInput(name) || !trimInput(symbol)) {
      setError('Enter an instrument name and symbol.')
      return
    }
    setSaving(true)
    setError('')
    try {
      const saved = await api.createInstrument({ name: trimInput(name), symbol: trimInput(symbol), kind, quoteCurrency })
      setName('')
      setSymbol('')
      onCreated(saved)
    } catch (error) {
      const uncertain = !(error instanceof ApiError) || error.status >= 500
      setError(`${errorMessage(error)}${uncertain ? ' Creation may have succeeded. Reload instruments and check before submitting again.' : ''}`)
    } finally {
      setSaving(false)
    }
  }

  return (
    <form className="investment-form" onSubmit={submit} noValidate aria-label="Add an instrument">
      <h3>Add an instrument</h3>
      <fieldset disabled={saving}>
        <div className="full-width">
          <label htmlFor="instrument-name">Instrument name</label>
          <input id="instrument-name" value={name} onChange={(event) => setName(event.target.value)} placeholder="e.g. Apple Inc." required aria-describedby={error ? 'instrument-error' : undefined} />
        </div>
        <div>
          <label htmlFor="instrument-symbol">Symbol</label>
          <input id="instrument-symbol" value={symbol} onChange={(event) => setSymbol(event.target.value)} placeholder="e.g. AAPL" required aria-describedby={error ? 'instrument-error' : undefined} />
        </div>
        <div>
          <label htmlFor="instrument-kind">Type</label>
          <select id="instrument-kind" value={kind} onChange={(event) => setKind(event.target.value as InstrumentKind)}>
            {instrumentKinds.map((value) => <option key={value} value={value}>{instrumentKindLabels[value]}</option>)}
          </select>
        </div>
        <div>
          <label htmlFor="instrument-currency">Quote currency</label>
          <select id="instrument-currency" value={quoteCurrency} onChange={(event) => setQuoteCurrency(event.target.value as Currency)}>
            {currencies.map((currency) => <option key={currency}>{currency}</option>)}
          </select>
        </div>
        <button>{saving ? 'Creating…' : 'Create instrument'}</button>
      </fieldset>
      <p className="muted">Prices use this quote currency. After creating an instrument, select an account to add a holding.</p>
      {error && <p id="instrument-error" role="alert" className="error">{error}</p>}
    </form>
  )
}

function InstrumentDetails({ id }: { id: string }) {
  const load = useCallback(() => api.getInstrument(id), [id])
  const details = useResource(load)
  return (
    <section className="instrument-details" aria-label="Instrument details">
      <div className="section-heading">
        <h3>{details.data?.name ?? 'Instrument details'}</h3>
        <button className="secondary" onClick={details.reload} disabled={details.loading}>Reload details</button>
      </div>
      {details.loading && <p role="status">Loading instrument…</p>}
      {details.error && <p role="alert" className="error">{details.error} Reload to try again.{details.data && ' Details below are from the last successful load.'}</p>}
      {details.data && <>
        <p><strong>{details.data.symbol}</strong> · {instrumentKindLabels[details.data.kind]} · {details.data.quoteCurrency}</p>
        <p className="account-id">{details.data.id}</p>
        <Prices instrument={details.data} />
      </>}
    </section>
  )
}

export function Instruments({ catalog }: { catalog: InstrumentCatalog }) {
  const [selectedId, setSelectedId] = useState<string>()
  const [search, setSearch] = useState('')
  const [notice, setNotice] = useState('')
  const query = search.trim().toLowerCase()
  const instruments = catalog.data?.instruments.filter((item) =>
    `${item.name} ${item.symbol} ${instrumentKindLabels[item.kind]} ${item.quoteCurrency} ${item.id}`.toLowerCase().includes(query))

  return (
    <section className="panel instruments" id="instruments" aria-label="Instruments and prices">
      <div className="section-heading">
        <h2>Instruments & prices</h2>
        <button className="secondary" onClick={catalog.reload} disabled={catalog.loading}>Reload instruments</button>
      </div>
      <p className="muted">A shared catalog for all your accounts. Select an instrument to view its details and record prices.</p>
      {catalog.loading && <p role="status">Loading instruments…</p>}
      {catalog.error && <p role="alert" className="error">{catalog.error} Reload to try again.{catalog.data && ' Instruments below are from the last successful load.'}</p>}
      <div className="instrument-workspace">
        <div>
          <label htmlFor="instrument-search">Find an instrument</label>
          <input id="instrument-search" type="search" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Search name, symbol, or currency" />
          {catalog.data?.instruments.length === 0 && <p className="empty">No instruments yet. Add your first one below.</p>}
          {!!catalog.data?.instruments.length && instruments?.length === 0 && <p className="empty">No matching instruments. Try another search.</p>}
          <ul className="account-list instrument-list" aria-label="Choose an instrument">
            {instruments?.map((item) => <li key={item.id}>
              <button className="account-button" aria-pressed={selectedId === item.id} onClick={() => setSelectedId(item.id)}>
                <strong>{item.symbol} · {item.name}</strong>
                <span className="muted">{instrumentKindLabels[item.kind]} · {item.quoteCurrency}</span>
                <span className="account-id">{item.id}</span>
              </button>
            </li>)}
          </ul>
          <CreateInstrument onCreated={(saved) => {
            setSelectedId(saved.id)
            setSearch('')
            setNotice(`Created ${saved.name}. It is now available when setting account holdings.`)
            catalog.reload()
          }} />
          {notice && <p role="status" className="success">{notice}</p>}
        </div>
        {selectedId ? <InstrumentDetails key={selectedId} id={selectedId} />
          : <div className="empty selection"><h3>Select an instrument</h3><p>View its quote currency and price history, or record a new price.</p></div>}
      </div>
    </section>
  )
}
