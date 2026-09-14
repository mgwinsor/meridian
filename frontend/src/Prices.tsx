import { useCallback, useState } from 'react'
import type { SubmitEvent } from 'react'
import { api, ApiError, errorMessage } from './api'
import type { Instrument } from './api'
import { trimInput, validateAmount } from './money'
import { observationTimestamp } from './investments'
import { useResource } from './useResource'

export function Prices({ instrument }: { instrument: Instrument }) {
  const load = useCallback(() => api.listPrices(instrument.id), [instrument.id])
  const history = useResource(load)
  const [amount, setAmount] = useState('')
  const [observedAt, setObservedAt] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone
  const observations = history.data?.observations
  const latest = observations?.at(-1)

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault()
    if (saving) return
    setNotice('')
    setError('')
    let timestamp: string | undefined
    try {
      const validation = validateAmount(amount, instrument.quoteCurrency)
      if (validation) throw new Error(validation)
      timestamp = observationTimestamp(observedAt)
    } catch (error) {
      setError(errorMessage(error))
      return
    }
    setSaving(true)
    try {
      const saved = await api.recordPrice(instrument.id, trimInput(amount), timestamp)
      setAmount('')
      setObservedAt('')
      setNotice(`Recorded ${saved.amount} ${saved.currency}.`)
      history.reload()
    } catch (error) {
      const uncertain = !(error instanceof ApiError) || error.status >= 500
      setError(`${errorMessage(error)}${uncertain ? ' Recording may have succeeded. Reload prices and check before submitting again to avoid duplicates.' : ''}`)
    } finally {
      setSaving(false)
    }
  }

  return (
    <section aria-label={`Prices for ${instrument.symbol}`}>
      <div className="section-heading">
        <h3>Price observations</h3>
        <button className="secondary" onClick={history.reload} disabled={history.loading}>Reload prices</button>
      </div>
      {history.loading && <p role="status">Loading prices…</p>}
      {history.error && <p role="alert" className="error">{history.error} Reload to try again.{history.data && ' Prices below are from the last successful load.'}</p>}
      {latest && <p className="price-highlight"><span className="muted">Latest observed price</span><strong>{latest.amount} {latest.currency}</strong><time dateTime={latest.observedAt}>{latest.observedAt.replace('T', ' ').replace('Z', ' UTC')}</time></p>}
      {observations?.length === 0 && <p className="empty">No prices recorded. Add an observation below to start the history.</p>}
      {!!observations?.length && <div className="table-scroll" tabIndex={0} role="region" aria-label="Price history">
        <table>
          <caption>Price history · newest first · UTC</caption>
          <thead><tr><th scope="col">Observed at</th><th scope="col">Price ({instrument.quoteCurrency})</th></tr></thead>
          <tbody>{observations.toReversed().map((observation, index) => <tr key={index}>
            <td><time dateTime={observation.observedAt}>{observation.observedAt.replace('T', ' ').replace('Z', '')}</time></td>
            <td>{observation.amount}</td>
          </tr>)}</tbody>
        </table>
      </div>}
      <form className="investment-form" onSubmit={submit} noValidate aria-label="Record a price">
        <h3>Record a price</h3>
        <p className="muted">Each save adds an observation, including repeated prices and times.</p>
        <fieldset disabled={saving}>
          <div>
            <label htmlFor="price-amount">Price ({instrument.quoteCurrency})</label>
            <input id="price-amount" inputMode={instrument.quoteCurrency === 'VND' ? 'numeric' : 'decimal'}
              value={amount} onChange={(event) => { setAmount(event.target.value); setNotice('') }}
              placeholder={instrument.quoteCurrency === 'VND' ? '0' : '0.00'} required
              aria-describedby={`price-help${error ? ' price-error' : ''}`} />
          </div>
          <div>
            <label htmlFor="price-time">Observed at (optional)</label>
            <input id="price-time" type="datetime-local" step="1" value={observedAt}
              onChange={(event) => { setObservedAt(event.target.value); setNotice('') }} aria-describedby={`price-time-help${error ? ' price-error' : ''}`} />
          </div>
          <button className="form-action">{saving ? 'Recording…' : 'Record price'}</button>
        </fieldset>
        <p id="price-help" className="muted">{instrument.quoteCurrency === 'VND' ? 'Whole VND only.' : 'Up to two decimal places.'} No commas. Zero is valid.</p>
        <p id="price-time-help" className="muted">Enter a time in {timezone}. Leave blank to use the server’s current time. History is shown in UTC.</p>
        {error && <p id="price-error" role="alert" className="error">{error}</p>}
        {notice && <p role="status" className="success">{notice}</p>}
      </form>
    </section>
  )
}
