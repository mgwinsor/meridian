import { afterAll, afterEach, beforeAll, beforeEach, expect, spyOn, test } from 'bun:test'
import { act } from 'react'
import type { Root } from 'react-dom/client'
import { Window } from 'happy-dom'
import { api, ApiError } from '../src/api'
import type { Instrument, InvestmentClass } from '../src/api'
import { Instruments } from '../src/Instruments'

const instrument: Instrument = {
  id: 'instrument-id',
  kind: 'stock',
  name: 'Example stock',
  symbol: 'EX',
  quoteCurrency: 'USD',
  assetClass: 'equities',
}

const browser = new Window()
const globals = {
  window: browser,
  document: browser.document,
  navigator: browser.navigator,
  IS_REACT_ACT_ENVIRONMENT: true,
}
const originalGlobals = new Map(
  Object.keys(globals).map((key) => [key, Object.getOwnPropertyDescriptor(globalThis, key)])
)
let createRoot: typeof import('react-dom/client').createRoot
let root: Root
let container: HTMLDivElement
const getInstrument = spyOn(api, 'getInstrument')
const setAssetClass = spyOn(api, 'setAssetClass')
const listPrices = spyOn(api, 'listPrices')

beforeAll(async () => {
  for (const [key, value] of Object.entries(globals)) {
    Object.defineProperty(globalThis, key, { configurable: true, writable: true, value })
  }
  createRoot = (await import('react-dom/client')).createRoot
})

beforeEach(() => {
  getInstrument.mockReset().mockImplementation(async () => ({ ...instrument }))
  setAssetClass.mockReset().mockImplementation(async (_id, assetClass) => ({
    ...instrument,
    assetClass,
  }))
  listPrices.mockReset().mockResolvedValue({ observations: [] })
  container = document.createElement('div')
  document.body.append(container)
  root = createRoot(container)
})

afterEach(async () => {
  await act(async () => root.unmount())
  container.remove()
})

afterAll(async () => {
  getInstrument.mockRestore()
  setAssetClass.mockRestore()
  listPrices.mockRestore()
  for (const [key, descriptor] of originalGlobals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor)
    else Reflect.deleteProperty(globalThis, key)
  }
  await browser.happyDOM.close()
})

async function openInstrument() {
  await act(async () => {
    root.render(
      <Instruments
        catalog={{
          data: { instruments: [instrument] },
          loading: false,
          reload: () => {},
        }}
      />
    )
  })
  await click('EX · Example stock')
}

async function click(label: string) {
  const button = Array.from(container.querySelectorAll('button')).find((item) =>
    item.textContent?.includes(label)
  )
  expect(button).toBeDefined()
  await act(async () => button!.click())
}

function classSelect() {
  return container.querySelector<HTMLSelectElement>('#edit-class')!
}

async function chooseClass(value: InvestmentClass) {
  await act(async () => {
    classSelect().value = value
    classSelect().dispatchEvent(new browser.Event('change', { bubbles: true }))
  })
}

for (const savedClass of ['equities', 'mixed'] as const) {
  test(`successful reload replaces a failed draft with saved class ${savedClass}`, async () => {
    await openInstrument()
    await chooseClass('bonds')
    setAssetClass.mockRejectedValueOnce(new ApiError(500, 'Save failed'))
    await click('Save class')
    expect(classSelect().value).toBe('bonds')
    expect(container.textContent).toContain('Reload details before saving again.')

    getInstrument.mockResolvedValueOnce({ ...instrument, assetClass: savedClass })
    await click('Reload details')
    expect(classSelect().value).toBe(savedClass)
    expect(container.textContent).not.toContain('Reload details before saving again.')

    await click('Save class')
    expect(setAssetClass).toHaveBeenLastCalledWith(instrument.id, savedClass)
    expect(container.textContent).toContain('Investment class saved.')
  })
}

test('failed reload preserves an unsaved classification draft', async () => {
  await openInstrument()
  await chooseClass('bonds')
  getInstrument.mockRejectedValueOnce(new Error('Read failed'))
  await click('Reload details')
  expect(container.textContent).toContain('Read failed')
  expect(classSelect().value).toBe('bonds')

  await click('Reload details')
  expect(classSelect().value).toBe('equities')
})
