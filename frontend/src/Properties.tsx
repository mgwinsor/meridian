import { useCallback, useId, useState } from 'react'
import type { SubmitEvent } from 'react'
import { api, ApiError, currencies, errorMessage } from './api'
import type { Currency, Property } from './api'
import { trimInput, validateAmount } from './money'
import { useResource } from './useResource'

function PropertyForm({ property, onSaved, onCancel }: {
  property?: Property
  onSaved: (property: Property) => void
  onCancel?: () => void
}) {
  const formId = useId()
  const [name, setName] = useState('')
  const [currency, setCurrency] = useState<Currency>(property?.value.currency ?? 'USD')
  const [amount, setAmount] = useState(property?.value.amount ?? '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault()
    if (saving) return
    const validation = !property && !trimInput(name) ? 'Enter a property name.' : validateAmount(amount, currency)
    if (validation) {
      setError(validation)
      return
    }
    setSaving(true)
    setError('')
    try {
      const value = { currency, amount: trimInput(amount) }
      const saved = property
        ? await api.setPropertyValue(property.id, value)
        : await api.createProperty(trimInput(name), value)
      setName('')
      setAmount('')
      onSaved(saved)
    } catch (error) {
      const uncertain = !(error instanceof ApiError) || error.status >= 500
      const guidance = property
        ? ' Reload properties to check the current value before saving again.'
        : uncertain ? ' Creation may have succeeded. Reload properties and check before submitting again.' : ''
      setError(`${errorMessage(error)}${guidance}`)
    } finally {
      setSaving(false)
    }
  }

  return (
    <form className="property-form" onSubmit={submit} noValidate aria-label={property ? `Update value for ${property.name}` : 'Add a property'}>
      <h3>{property ? 'Update current value' : 'Add a property'}</h3>
      {property && <p className="muted">Saving replaces the currency and amount. Changing currency requires a new estimate; no conversion is performed.</p>}
      <fieldset disabled={saving}>
        {!property && <div className="property-name-field">
          <label htmlFor={`${formId}-name`}>Property name</label>
          <input id={`${formId}-name`} value={name} onChange={(event) => setName(event.target.value)}
            placeholder="e.g. Singapore home" required aria-describedby={error ? `${formId}-error` : undefined} />
        </div>}
        <div>
          <label htmlFor={`${formId}-currency`}>Currency</label>
          <select id={`${formId}-currency`} value={currency} onChange={(event) => {
            setCurrency(event.target.value as Currency)
            setError('')
          }}>
            {currencies.map((code) => <option key={code}>{code}</option>)}
          </select>
        </div>
        <div>
          <label htmlFor={`${formId}-amount`}>Current value</label>
          <input id={`${formId}-amount`} type="text" inputMode={currency === 'VND' ? 'numeric' : 'decimal'}
            value={amount} onChange={(event) => setAmount(event.target.value)}
            placeholder={currency === 'VND' ? '0' : '0.00'} required
            aria-describedby={`${formId}-help${error ? ` ${formId}-error` : ''}`} />
        </div>
        <button>{saving ? 'Saving…' : property ? 'Save value' : 'Create property'}</button>
      </fieldset>
      <p id={`${formId}-help`} className="muted">{currency === 'VND' ? 'Whole VND only.' : 'Up to two decimal places.'} Enter amounts without commas. Zero is a valid value.</p>
      {error && <p id={`${formId}-error`} role="alert" className="error">{error}</p>}
      {onCancel && <button type="button" className="secondary" onClick={onCancel} disabled={saving}>Cancel</button>}
    </form>
  )
}

export function Properties() {
  const load = useCallback(() => api.listProperties(), [])
  const properties = useResource(load)
  const [editingId, setEditingId] = useState<string>()
  const [notice, setNotice] = useState('')

  return (
    <section className="panel properties" aria-label="Properties" aria-busy={properties.loading}>
      <div className="section-heading">
        <h2>Properties</h2>
        <button className="secondary" onClick={properties.reload} disabled={properties.loading}>Reload properties</button>
      </div>
      <p className="muted">Properties are owned directly in this workspace, independently of accounts. Enter the estimated gross market value of your owned share, before mortgages or other liabilities.</p>
      {properties.loading && <p role="status">Loading properties…</p>}
      {properties.error && <p role="alert" className="error">{properties.error} Reload to try again.{properties.data && ' Properties below are from the last successful load.'}</p>}
      {properties.data && <>
        {properties.data.properties.length === 0
          ? <p className="empty">No properties yet. Add your first property below.</p>
          : <ul className="property-list">
            {properties.data.properties.map((property) => <li className="property-card" key={property.id}>
              <h4>{property.name}</h4>
              <p className="account-id">{property.id}</p>
              <p className="property-value">{property.value.amount} {property.value.currency}</p>
              {editingId === property.id
                ? <PropertyForm property={property} onCancel={() => setEditingId(undefined)} onSaved={(saved) => {
                  setEditingId(undefined)
                  setNotice(`Saved ${saved.name}: ${saved.value.amount} ${saved.value.currency}.`)
                  properties.reload()
                }} />
                : <button className="secondary" aria-label={`Update value for ${property.name} (${property.id})`} onClick={() => {
                  setEditingId(property.id)
                  setNotice('')
                }}>Update value</button>}
            </li>)}
          </ul>}
        <PropertyForm onSaved={(saved) => {
          setNotice(`Created ${saved.name}.`)
          properties.reload()
        }} />
      </>}
      {notice && <p role="status" className="success">{notice}</p>}
    </section>
  )
}
