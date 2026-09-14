import { useCallback, useState } from 'react'
import type { SubmitEvent } from 'react'
import { api, errorMessage } from './api'
import type { InstrumentCatalog } from './Instruments'
import { trimInput } from './money'
import { validateQuantity } from './investments'
import { useResource } from './useResource'

export function Positions({ accountId, catalog }: { accountId: string; catalog: InstrumentCatalog }) {
  const load = useCallback(() => api.listPositions(accountId), [accountId])
  const holdings = useResource(load)
  const [instrumentId, setInstrumentId] = useState('')
  const [quantity, setQuantity] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const instruments = catalog.data?.instruments ?? []
  const selected = instruments.find((item) => item.id === instrumentId)
  const current = holdings.data?.positions.find((item) => item.instrumentId === instrumentId)

  function selectInstrument(id: string) {
    setInstrumentId(id)
    setQuantity(holdings.data?.positions.find((item) => item.instrumentId === id)?.quantity ?? '')
    setError('')
    setNotice('')
  }

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault()
    if (saving) return
    setNotice('')
    const validation = !selected ? 'Choose an instrument for this holding.' : validateQuantity(quantity)
    if (validation) {
      setError(validation)
      return
    }
    setSaving(true)
    setError('')
    try {
      const saved = await api.setPosition(accountId, instrumentId, trimInput(quantity))
      setQuantity(saved.quantity)
      setNotice(`Saved ${saved.quantity} units of ${selected?.symbol}.`)
      holdings.reload()
    } catch (error) {
      setError(`${errorMessage(error)} Reload holdings to check the current quantity before saving again.`)
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="positions" aria-label="Account holdings">
      <div className="section-heading">
        <h3>Holdings</h3>
        <button className="secondary" onClick={holdings.reload} disabled={holdings.loading}>Reload holdings</button>
      </div>
      {holdings.loading && <p role="status">Loading holdings…</p>}
      {holdings.error && <p role="alert" className="error">{holdings.error} Reload to try again.{holdings.data && ' Holdings below are from the last successful load.'}</p>}
      {holdings.data?.positions.length === 0 && <p className="empty">No holdings in this account. Choose an instrument and enter the units you own.</p>}
      {!!holdings.data?.positions.length && <ul className="holding-list">
        {holdings.data.positions.map((position) => {
          const instrument = instruments.find((item) => item.id === position.instrumentId)
          return <li className="holding-card" key={position.instrumentId}>
            <div>
              <strong>{instrument ? `${instrument.symbol} · ${instrument.name}` : 'Instrument unavailable'}</strong>
              <span className="account-id">{position.instrumentId}</span>
              <p className="holding-quantity">{position.quantity} <span className="muted">units</span></p>
            </div>
            <button className="secondary" disabled={saving || !instrument} aria-label={`Edit holding for ${instrument?.symbol ?? position.instrumentId} (${position.instrumentId})`}
              onClick={() => { selectInstrument(position.instrumentId); document.getElementById('holding-quantity')?.focus() }}>Edit holding</button>
          </li>
        })}
      </ul>}
      {catalog.loading && <p role="status">Loading available instruments…</p>}
      {catalog.error && <p role="alert" className="error">{catalog.error} {catalog.data ? 'Using the last loaded instrument catalog. ' : ''}<button className="secondary" onClick={catalog.reload} disabled={catalog.loading}>Reload instruments</button></p>}
      {catalog.data?.instruments.length === 0 && <p className="empty"><a href="#instruments">Add an instrument</a> to the shared catalog before setting a holding.</p>}
      {!!instruments.length && <form className="investment-form" onSubmit={submit} noValidate aria-label="Set a holding">
        <h3>Set a holding</h3>
        <p className="muted">Saving replaces the total units held in this account. Cash stays the same. Set zero to keep a visible zero holding.</p>
        <fieldset disabled={saving}>
          <div className="full-width">
            <label htmlFor="holding-instrument">Instrument</label>
            <select id="holding-instrument" value={instrumentId} onChange={(event) => selectInstrument(event.target.value)} required aria-describedby={error ? 'holding-error' : undefined}>
              <option value="">Choose an instrument</option>
              {instruments.map((item) => <option key={item.id} value={item.id}>{item.symbol} · {item.name} · {item.quoteCurrency} · {item.id}</option>)}
            </select>
          </div>
          <div>
            <label htmlFor="holding-quantity">Quantity (units)</label>
            <input id="holding-quantity" inputMode="decimal" value={quantity} onChange={(event) => { setQuantity(event.target.value); setNotice('') }}
              placeholder="e.g. 12.5" required aria-describedby={`holding-help${error ? ' holding-error' : ''}`} />
          </div>
          <button>{saving ? 'Saving…' : 'Save holding'}</button>
        </fieldset>
        {selected && <p className="muted">Last loaded quantity: {current?.quantity ?? 'No holding yet'} · Quote currency: {selected.quoteCurrency}</p>}
        <p id="holding-help" className="muted">Fractional units are allowed at any decimal precision. Enter quantities without commas.</p>
        {error && <p id="holding-error" role="alert" className="error">{error}</p>}
        {notice && <p role="status" className="success">{notice}</p>}
      </form>}
    </section>
  )
}
