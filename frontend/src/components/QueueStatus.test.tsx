import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen } from '../test/utils'
import { QueueStatus } from './QueueStatus'
import { useAppStore } from '../store'
import type { Queue } from '../types'

describe('QueueStatus', () => {
  beforeEach(() => {
    useAppStore.setState({ queue: null })
  })

  it('renders queue statistics correctly', () => {
    const queue: Queue = {
      total: 15,
      active: 10,
      deferred: 5,
    }

    useAppStore.setState({ queue })
    render(<QueueStatus />)

    expect(screen.getByText('Queue: 10 active, 5 deferred, 15 total')).toBeInTheDocument()
  })

  it('handles zero values correctly', () => {
    const queue: Queue = {
      total: 0,
      active: 0,
      deferred: 0,
    }

    useAppStore.setState({ queue })
    render(<QueueStatus />)

    expect(screen.getByText('Queue: 0 active, 0 total')).toBeInTheDocument()
  })

  it('handles single item queue', () => {
    const queue: Queue = {
      total: 1,
      active: 1,
      deferred: 0,
    }

    useAppStore.setState({ queue })
    render(<QueueStatus />)

    expect(screen.getByText('Queue: 1 active, 1 total')).toBeInTheDocument()
  })

  it('handles all deferred items', () => {
    const queue: Queue = {
      total: 3,
      active: 0,
      deferred: 3,
    }

    useAppStore.setState({ queue })
    render(<QueueStatus />)

    expect(screen.getByText('Queue: 0 active, 3 deferred, 3 total')).toBeInTheDocument()
  })

  it('renders null when no queue data', () => {
    useAppStore.setState({ queue: null })
    const { container } = render(<QueueStatus />)

    expect(container.firstChild).toBeNull()
  })

  it('has correct styling classes', () => {
    const queue: Queue = {
      total: 5,
      active: 3,
      deferred: 2,
    }

    useAppStore.setState({ queue })
    const { container } = render(<QueueStatus />)
    const queueElement = container.firstChild as HTMLElement

    expect(queueElement).toHaveClass('flex', 'items-center', 'gap-2')
  })
})