import { useCallback, useState, useSyncExternalStore } from 'react'
import { api, currencies, valuationChanges } from './api'
import type { Breakdown, Currency, OverviewReport } from './api'
import { useResource } from './useResource'

function BreakdownTable({
  title,
  rows,
  currency,
}: {
  title: string
  rows: Breakdown[]
  currency: Currency
}) {
  return (
    <table>
      <caption>{title}</caption>
      <thead>
        <tr>
          <th scope="col">Group</th>
          <th scope="col">Value ({currency})</th>
          <th scope="col">% of valued subtotal</th>
          <th scope="col">Excluded assets</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((row) => (
          <tr key={row.key}>
            <th scope="row">{row.key}</th>
            <td>{row.amount}</td>
            <td>{row.percentage === undefined ? '—' : `${row.percentage}%`}</td>
            <td>{row.excludedAssets}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function Report({ currency }: { currency: Currency }) {
  const load = useCallback(() => api.overview(currency), [currency])
  const report = useResource(load)
  const data = report.data
  return (
    <div aria-busy={report.loading}>
      <button className="secondary" onClick={report.reload} disabled={report.loading}>
        Reload overview
      </button>
      {report.loading && <p role="status">Loading overview…</p>}
      {report.error && (
        <p role="alert" className="error">
          {report.error} Reload to try again.
          {data && ' Values below are from the last successful load.'}
        </p>
      )}
      {data && <OverviewValues data={data} />}
    </div>
  )
}

export function OverviewValues({ data }: { data: OverviewReport }) {
  return (
    <>
      <p className="overview-total">
        {data.complete ? 'Total assets' : 'Valued subtotal'}:{' '}
        <strong>
          {data.valuedTotal} {data.currency}
        </strong>
      </p>
      {!data.complete && (
        <p role="status" className="workspace-note">
          Incomplete valuation. Assets with missing prices or FX rates are excluded. Add the inputs
          listed below to complete the total.
        </p>
      )}
      <p className="muted">
        Generated <time dateTime={data.generatedAt}>{data.generatedAt}</time>. Uses the latest
        observations at or before this time, without automatic expiry.
      </p>
      {data.assets.length === 0 && (
        <p className="empty">No assets yet. Add cash, a holding, or a property below.</p>
      )}
      <div className="table-scroll" tabIndex={0} role="region" aria-label="Asset class breakdown">
        <BreakdownTable title="Asset classes" rows={data.assetClasses} currency={data.currency} />
      </div>
      <div className="table-scroll" tabIndex={0} role="region" aria-label="Currency breakdown">
        <BreakdownTable
          title="Currency distribution"
          rows={data.currencies}
          currency={data.currency}
        />
      </div>
      <p className="muted">
        Currency distribution groups cash and property denominations and investment quote
        currencies. It does not describe geographic exposure. Percentages may not sum to 100% after
        rounding.
      </p>
      <details>
        <summary>Valuation details ({data.assets.length} assets)</summary>
        <div className="table-scroll" tabIndex={0} role="region" aria-label="Asset valuations">
          <table>
            <caption>Each source is counted once. Converted values are rounded per asset.</caption>
            <thead>
              <tr>
                <th scope="col">Asset / source</th>
                <th scope="col">Class</th>
                <th scope="col">Native value</th>
                <th scope="col">Selected price / units</th>
                <th scope="col">Value ({data.currency})</th>
                <th scope="col">Missing inputs</th>
              </tr>
            </thead>
            <tbody>
              {data.assets.map((asset) => (
                <tr key={`${asset.source}:${asset.id}`}>
                  <th scope="row">
                    {asset.name}
                    <br />
                    <small>
                      {asset.source} · {asset.id}
                    </small>
                  </th>
                  <td>{asset.assetClass}</td>
                  <td>
                    {asset.nativeValue ?? '—'} {asset.currency}
                  </td>
                  <td>
                    {asset.quantity !== undefined && (
                      <span>
                        {asset.quantity} units
                        <br />
                      </span>
                    )}
                    {asset.price
                      ? `${asset.price.amount} ${asset.price.currency} at ${asset.price.observedAt}`
                      : '—'}
                  </td>
                  <td>{asset.convertedValue ?? 'Excluded'}</td>
                  <td>{asset.missingInputs.join(', ') || 'None'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <h3>FX observations used</h3>
        <p>Rates are currency units per USD. USD equals 1.</p>
        <ul>
          {data.fxObservations.map((rate) => (
            <li key={rate.currency}>
              {rate.currency}: {rate.rate} at {rate.observedAt}
            </li>
          ))}
        </ul>
        {data.fxObservations.length === 0 && <p>No FX observations required.</p>}
      </details>
    </>
  )
}

export function Overview() {
  const [currency, setCurrency] = useState<Currency>('USD')
  const revision = useSyncExternalStore(valuationChanges.subscribe, valuationChanges.getSnapshot)
  return (
    <section className="panel overview" aria-label="Total assets overview">
      <h2>Total assets overview</h2>
      <label htmlFor="reporting-currency">Reporting currency</label>
      <select
        id="reporting-currency"
        value={currency}
        onChange={(event) => setCurrency(event.target.value as Currency)}
      >
        {currencies.map((code) => (
          <option key={code}>{code}</option>
        ))}
      </select>
      {/* Remount discards prior-currency data immediately and cancels late read results. */}
      <Report key={`${currency}:${revision}`} currency={currency} />
    </section>
  )
}
