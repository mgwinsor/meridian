import { useEffect, useState } from 'react'
import { errorMessage } from './api'

export function useResource<T>(load: () => Promise<T>) {
  const [state, setState] = useState<{ data?: T; loading: boolean; error?: string }>({ loading: true })
  const [revision, setRevision] = useState(0)

  useEffect(() => {
    let active = true
    load().then(
      (data) => { if (active) setState({ data, loading: false }) },
      (error: unknown) => {
        if (active) setState((previous) => ({ ...previous, loading: false, error: errorMessage(error) }))
      },
    )
    return () => { active = false }
  }, [load, revision])

  function reload() {
    setState((previous) => ({ ...previous, loading: true, error: undefined }))
    setRevision((previous) => previous + 1)
  }

  return { ...state, reload }
}
