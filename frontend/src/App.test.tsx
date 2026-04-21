import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import { runQuery } from './lib/neo4j'

vi.mock('./lib/neo4j')

const mockedRunQuery = vi.mocked(runQuery)

describe('App', () => {
  beforeEach(() => {
    mockedRunQuery.mockReset()
  })

  it('renders the heading, default query textarea, and an enabled Run button', () => {
    render(<App />)

    expect(screen.getByRole('heading', { name: /go call analysis/i })).toBeInTheDocument()

    const textarea = screen.getByRole('textbox')
    expect(textarea).toHaveValue('MATCH (n) RETURN n LIMIT 25')

    expect(screen.getByRole('button', { name: /run/i })).toBeEnabled()
  })

  it('renders a results table after clicking Run when the query succeeds', async () => {
    mockedRunQuery.mockResolvedValueOnce({
      records: [
        {
          keys: ['n'],
          get: (_key: string) => 'Alice',
        },
      ],
    } as never)

    const user = userEvent.setup()
    render(<App />)

    await user.click(screen.getByRole('button', { name: /run/i }))

    await waitFor(() => {
      expect(screen.getByRole('table')).toBeInTheDocument()
    })

    expect(screen.getByRole('columnheader', { name: 'n' })).toBeInTheDocument()
    expect(screen.getByRole('cell', { name: 'Alice' })).toBeInTheDocument()
  })

  it('renders an error message after clicking Run when the query fails', async () => {
    mockedRunQuery.mockRejectedValueOnce(new Error('Connection refused'))

    const user = userEvent.setup()
    render(<App />)

    await user.click(screen.getByRole('button', { name: /run/i }))

    await waitFor(() => {
      expect(screen.getByText('Connection refused')).toBeInTheDocument()
    })

    expect(screen.getByText('Connection refused')).toHaveClass('error')
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })
})
