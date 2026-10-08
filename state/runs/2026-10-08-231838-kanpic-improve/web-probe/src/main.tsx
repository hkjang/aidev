import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { App } from './App'
import './styles.css'
import './auth.css'
import './collaboration.css'
import './version-history.css'
import './sheet-tabs.css'
import './conflicts.css'
import './ai.css'

const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 15_000, retry: 1 }, mutations: { retry: 0 } } })

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}><App /></QueryClientProvider>
  </StrictMode>,
)
