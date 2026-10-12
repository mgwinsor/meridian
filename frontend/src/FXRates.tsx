import { useCallback, useState } from 'react'
import type { SubmitEvent } from 'react'
import { api, ApiError, errorMessage } from './api'
import { observationTimestamp, validateQuantity } from './investments'
import { useResource } from './useResource'

function RateHistory({ currency }: { currency: 'SGD' | 'VND' }) {
  const load = useCallback(() => api.listFX(currency), [currency])
  const history = useResource(load)
  const [rate, setRate] = useState('')
  const [observedAt, setObservedAt] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault()
    if (saving) return
    setNotice('')
    setError('')
    let timestamp: string | undefined
    try {
      const invalid = validateQuantity(rate)
      if (invalid || !/[1-9]/.test(rate)) {
        throw new Error(
          'Enter a positive rate with up to 20 integer and 18 fractional digits, ' +
            'without signs, commas, or exponents.'
        )
      }
      timestamp = observationTimestamp(observedAt)
    } catch (error) {
      setError(errorMessage(error))
      return
    }
    setSaving(true)
    try {
      await api.recordFX(currency, rate.trim(), timestamp)
      setRate('')
      setObservedAt('')
      setNotice('FX observation recorded.')
      history.reload()
    } catch (error) {
      const uncertain = !(error instanceof ApiError) || error.status >= 500
      const recovery = uncertain
        ? ' Recording may have succeeded. Reload history before submitting again to avoid duplicates.'
        : ''
      setError(`${errorMessage(error)}${recovery}`)
    } finally {
      setSaving(false)
    }
  }
  return (
    <>
      <button className="secondary" onClick={history.reload} disabled={history.loading}>
        Reload FX history
      </button>
      {history.loading && <p role="status">Loading FX history…</p>}
      {history.error && (
        <p role="alert" className="error">
          {history.error} Reload to try again.
        </p>
      )}
      {history.data?.observations.length === 0 && <p>No FX observations yet.</p>}
      <table>
        <caption>{currency} units per USD · newest first · UTC</caption>
        <thead>
          <tr>
            <th scope="col">Observed at</th>
            <th scope="col">Rate</th>
          </tr>
        </thead>
        <tbody>
          {history.data?.observations.toReversed().map((item, index) => (
            <tr key={index}>
              <td>{item.observedAt}</td>
              <td>{item.rate}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <form className="investment-form" onSubmit={submit} noValidate aria-label="Record an FX rate">
        <fieldset disabled={saving}>
          <div>
            <label htmlFor="fx-rate">{currency} units per 1 USD</label>
            <input
              id="fx-rate"
              inputMode="decimal"
              value={rate}
              onChange={(event) => setRate(event.target.value)}
              required
              aria-describedby="fx-help"
            />
          </div>
          <div>
            <label htmlFor="fx-time">Observed at (optional)</label>
            <input
              id="fx-time"
              type="datetime-local"
              step="1"
              value={observedAt}
              onChange={(event) => setObservedAt(event.target.value)}
              aria-describedby="fx-help"
            />
          </div>
          <button>{saving ? 'Recording…' : 'Record FX rate'}</button>
        </fieldset>
        <p id="fx-help" className="muted">
          Up to 18 fractional digits, including for VND rates. Time is in{' '}
          {Intl.DateTimeFormat().resolvedOptions().timeZone}; leave blank for server time. Each save
          adds an observation.
        </p>
        {error && (
          <p role="alert" className="error">
            {error}
          </p>
        )}
        {notice && (
          <p role="status" className="success">
            {notice}
          </p>
        )}
      </form>
    </>
  )
}

export function FXRates() {
  const [currency, setCurrency] = useState<'SGD' | 'VND'>('SGD')
  return (
    <section className="panel fx-rates" aria-label="FX rates">
      <h2>FX rates</h2>
      <p>Manual currency units per USD. USD always equals 1.</p>
      <label htmlFor="fx-currency">Quote currency</label>
      <select
        id="fx-currency"
        value={currency}
        onChange={(event) => setCurrency(event.target.value as 'SGD' | 'VND')}
      >
        <option>SGD</option>
        <option>VND</option>
      </select>
      <RateHistory key={currency} currency={currency} />
    </section>
  )
}
