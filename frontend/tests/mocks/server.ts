import { setupServer } from 'msw/node'
import { handlers } from './handlers'

/**
 * MSW server instance shared across all test suites.
 * Lifecycle (listen / resetHandlers / close) is managed in tests/setup.ts.
 */
export const server = setupServer(...handlers)
