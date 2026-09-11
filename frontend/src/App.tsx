import { useCallback, useState } from 'react'
import type { SubmitEvent } from 'react'
import { api, ApiError, currencies, errorMessage } from './api'
import type { Account, CashBalance, Currency } from './api'
import { trimInput, validateAmount } from './money'
import { useResource } from './useResource'
import './App.css'

function ConnectionStatus() {
  const health = useResource(api.health)
  return (
    <div className="connection">
      <span role="status" className={health.error ? 'status offline' : 'status'}>
        {health.loading ? 'Checking backend…' : health.error ? 'Backend unavailable' : 'Backend connected'}
      </span>
      <button className="secondary" onClick={health.reload} disabled={health.loading}>Check connection</button>
      {health.error && <p role="alert" className="error">{health.error}</p>}
    </div>
  )
}

function CreateAccount({ onCreated }: { onCreated: (account: Account) => void }) {
  const [name, setName] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault()
    if (saving) return
    if (!trimInput(name)) {
      setError('Enter an account name.')
      return
    }
    setSaving(true)
    setError('')
    try {
      const account = await api.createAccount(trimInput(name))
      setName('')
      onCreated(account)
    } catch (error) {
      const uncertain = !(error instanceof ApiError) || error.status >= 500
      setError(`${errorMessage(error)}${uncertain ? ' Creation may have succeeded. Reload accounts and check before submitting again.' : ''}`)
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={submit} className="create-form" noValidate>
      <h3>Add an account</h3>
      <label htmlFor="account-name">Account name</label>
      <input id="account-name" value={name} onChange={(event) => setName(event.target.value)}
        placeholder="e.g. HSBC Premier" disabled={saving} required
        aria-invalid={!!error} aria-describedby={error ? 'account-error' : undefined} />
      {error && <p id="account-error" role="alert" className="error">{error}</p>}
      <button disabled={saving}>{saving ? 'Creating…' : 'Create account'}</button>
    </form>
  )
}

function CashForm({ accountId, balances, onSaved }: {
  accountId: string
  balances: CashBalance[]
  onSaved: () => void
}) {
  const [currency, setCurrency] = useState<Currency>('USD')
  const [amount, setAmount] = useState(() => balances.find((balance) => balance.currency === 'USD')?.amount ?? '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault()
    if (saving) return
    setNotice('')
    const validation = validateAmount(amount, currency)
    if (validation) {
      setError(validation)
      return
    }
    setSaving(true)
    setError('')
    try {
      const saved = await api.setCash(accountId, currency, trimInput(amount))
      setAmount(saved.amount)
      setNotice(`Saved ${saved.amount} ${saved.currency}.`)
      onSaved()
    } catch (error) {
      setError(`${errorMessage(error)} Reload balances to check the current amount before saving again.`)
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={submit} className="cash-form" noValidate>
      <h3>Set cash balance</h3>
      <p className="muted">Saving replaces the current balance for this currency. Zero is a valid balance.</p>
      <fieldset disabled={saving}>
        <div>
          <label htmlFor="currency">Currency</label>
          <select id="currency" value={currency} onChange={(event) => {
            const next = event.target.value as Currency
            setCurrency(next)
            setAmount(balances.find((balance) => balance.currency === next)?.amount ?? '')
            setError('')
            setNotice('')
          }}>
            {currencies.map((code) => <option key={code}>{code}</option>)}
          </select>
        </div>
        <div>
          <label htmlFor="amount">Amount</label>
          <input id="amount" type="text" inputMode={currency === 'VND' ? 'numeric' : 'decimal'}
            value={amount} onChange={(event) => { setAmount(event.target.value); setNotice('') }}
            placeholder={currency === 'VND' ? '0' : '0.00'} required aria-invalid={!!error}
            aria-describedby={`amount-help${error ? ' cash-error' : ''}`} />
        </div>
        <button>{saving ? 'Saving…' : 'Save balance'}</button>
      </fieldset>
      <p id="amount-help" className="muted">{currency === 'VND' ? 'Whole VND only.' : 'Up to two decimal places.'} Enter amounts without commas.</p>
      {error && <p id="cash-error" role="alert" className="error">{error}</p>}
      {notice && <p role="status" className="success">{notice}</p>}
    </form>
  )
}

function AccountDetails({ id }: { id: string }) {
  const load = useCallback(async () => {
    const [account, cash] = await Promise.all([api.getAccount(id), api.listCash(id)])
    return { account, balances: cash.balances }
  }, [id])
  const details = useResource(load)

  return (
    <section className="panel details" aria-label="Account details" aria-busy={details.loading}>
      <div className="section-heading">
        <h2>{details.data?.account.name ?? 'Account details'}</h2>
        <button className="secondary" onClick={details.reload} disabled={details.loading}>Reload balances</button>
      </div>
      <p className="account-id">{id}</p>
      {details.loading && <p role="status">Loading account and balances…</p>}
      {details.error && <p role="alert" className="error">{details.error} Reload to try again.{details.data && ' Balances below are from the last successful load.'}</p>}
      {details.data && <>
        <h3>Current cash</h3>
        {details.data.balances.length === 0
          ? <p className="empty">No cash balances yet. Set your first balance below.</p>
          : <table>
            <caption className="sr-only">Current cash balances</caption>
            <thead><tr><th scope="col">Currency</th><th scope="col">Amount</th></tr></thead>
            <tbody>{details.data.balances.map((balance) => (
              <tr key={balance.currency}><th scope="row">{balance.currency}</th><td>{balance.amount}</td></tr>
            ))}</tbody>
          </table>}
        <CashForm accountId={id} balances={details.data.balances} onSaved={details.reload} />
      </>}
    </section>
  )
}

function App() {
  const accounts = useResource(api.listAccounts)
  const [selectedId, setSelectedId] = useState<string>()
  const [notice, setNotice] = useState('')

  return (
    <main>
      <header>
        <div><p className="eyebrow">Personal workspace</p><h1>Meridian</h1><p className="muted">Your accounts and current cash balances.</p></div>
        <ConnectionStatus />
      </header>
      <p className="workspace-note">This development workspace stores data in memory. Restarting the backend clears all accounts and balances.</p>
      <div className="workspace">
        <section className="panel accounts" aria-label="Accounts">
          <div className="section-heading">
            <h2>Accounts</h2>
            <button className="secondary" onClick={accounts.reload} disabled={accounts.loading}>Reload accounts</button>
          </div>
          {accounts.loading && <p role="status">Loading accounts…</p>}
          {accounts.error && <p role="alert" className="error">{accounts.error} Reload to try again.</p>}
          {accounts.data?.accounts.length === 0 && <p className="empty">No accounts yet. Add one to get started.</p>}
          <ul className="account-list" aria-label="Choose an account">
            {accounts.data?.accounts.map((account) => (
              <li key={account.id}>
                <button className="account-button" aria-pressed={selectedId === account.id} onClick={() => {
                  setSelectedId(account.id)
                  setNotice('')
                }}>
                  <strong>{account.name}</strong><span className="account-id">{account.id}</span>
                </button>
              </li>
            ))}
          </ul>
          <CreateAccount onCreated={(account) => {
            setSelectedId(account.id)
            setNotice(`Created ${account.name}.`)
            accounts.reload()
          }} />
          {notice && <p role="status" className="success">{notice}</p>}
        </section>
        {selectedId
          ? <AccountDetails key={selectedId} id={selectedId} />
          : <section className="panel empty selection"><h2>Select an account</h2><p>Choose an account to view or update its cash balances.</p></section>}
      </div>
      <footer>Balances are shown in their original currency. No currency conversion or combined total.</footer>
    </main>
  )
}

export default App
