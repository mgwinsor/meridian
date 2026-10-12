import { expect, test } from 'bun:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { OverviewValues } from '../src/Overview'
import type { OverviewReport } from '../src/api'

const empty: OverviewReport = {
  currency: 'USD',
  generatedAt: '2026-01-01T00:00:00Z',
  complete: true,
  valuedTotal: '0.00',
  assets: [],
  assetClasses: [],
  currencies: [],
  fxObservations: [],
}

test('empty overview explains how to add assets and provides accessible breakdown tables', () => {
  const html = renderToStaticMarkup(<OverviewValues data={empty} />)
  expect(html).toContain('No assets yet')
  expect(html).toContain('<caption>Asset classes</caption>')
  expect(html).toContain('<caption>Currency distribution</caption>')
  expect(html).toContain('scope="col"')
  expect(html).toContain('No FX observations required')
})

test('incomplete report labels a subtotal, exclusions and missing inputs without losing exact values', () => {
  const html = renderToStaticMarkup(
    <OverviewValues
      data={{
        ...empty,
        complete: false,
        valuedTotal: '184467440737095516.14',
        assets: [
          {
            source: 'position',
            id: 'account:instrument',
            name: 'Unpriced fund',
            assetClass: 'unclassified',
            currency: 'SGD',
            quantity: '1.5',
            missingInputs: ['price'],
          },
        ],
        assetClasses: [
          { key: 'unclassified', amount: '0.00', percentage: '0.00', excludedAssets: 1 },
        ],
      }}
    />
  )
  expect(html).toContain('Valued subtotal')
  expect(html).toContain('184467440737095516.14 USD')
  expect(html).toContain('Incomplete valuation')
  expect(html).toContain('Excluded assets')
  expect(html).toContain('Unpriced fund')
  expect(html).toContain('<td>Excluded</td><td>price</td>')
  expect(html).toContain('0.00%')
})

test('zero subtotal omits percentages and report denominations follow the supplied currency', () => {
  for (const currency of ['USD', 'SGD', 'VND'] as const) {
    const html = renderToStaticMarkup(
      <OverviewValues
        data={{
          ...empty,
          currency,
          valuedTotal: currency === 'VND' ? '0' : '0.00',
          currencies: [
            { key: 'SGD', amount: currency === 'VND' ? '0' : '0.00', excludedAssets: 0 },
          ],
        }}
      />
    )
    expect(html).toContain(`Value (${currency})`)
    expect(html).toContain('<td>—</td>')
    expect(html).not.toContain('Incomplete valuation')
  }
})
