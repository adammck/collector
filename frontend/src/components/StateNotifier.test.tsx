import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen } from '../test/utils'
import { StateNotifier } from './StateNotifier'
import { useAppStore } from '../store'
import type { AppState } from '../types'

describe('StateNotifier', () => {
  beforeEach(() => {
    useAppStore.setState({ state: 'idle', stateMessage: null })
  })

  it('renders idle state correctly', () => {
    useAppStore.setState({ state: 'idle' })
    render(<StateNotifier />)
    
    expect(screen.getByText('⏸️')).toBeInTheDocument()
    expect(screen.getByText('Idle')).toBeInTheDocument()
  })

  it('renders awaiting_data state correctly', () => {
    useAppStore.setState({ state: 'awaiting_data' })
    render(<StateNotifier />)
    
    expect(screen.getByText('⏳')).toBeInTheDocument()
    expect(screen.getByText('Waiting for data...')).toBeInTheDocument()
  })

  it('renders waiting_user state correctly', () => {
    useAppStore.setState({ state: 'waiting_user' })
    render(<StateNotifier />)
    
    expect(screen.getByText('✅')).toBeInTheDocument()
    expect(screen.getByText('Ready for response')).toBeInTheDocument()
  })

  it('renders submitting state correctly', () => {
    useAppStore.setState({ state: 'submitting' })
    render(<StateNotifier />)
    
    expect(screen.getByText('📤')).toBeInTheDocument()
    expect(screen.getByText('Submitting...')).toBeInTheDocument()
  })

  it('renders server_error state correctly', () => {
    useAppStore.setState({ state: 'server_error' })
    render(<StateNotifier />)
    
    expect(screen.getByText('⚠️')).toBeInTheDocument()
    expect(screen.getByText('Server error')).toBeInTheDocument()
  })

  it('renders client_error state correctly', () => {
    useAppStore.setState({ state: 'client_error' })
    render(<StateNotifier />)
    
    expect(screen.getByText('⚠️')).toBeInTheDocument()
    expect(screen.getByText('Client error')).toBeInTheDocument()
  })

  it('displays custom message when provided', () => {
    useAppStore.setState({ state: 'server_error', stateMessage: 'connection timeout' })
    render(<StateNotifier />)
    
    expect(screen.getByText('⚠️')).toBeInTheDocument()
    expect(screen.getByText('connection timeout')).toBeInTheDocument()
  })

  it('uses default message when no custom message', () => {
    useAppStore.setState({ state: 'waiting_user', stateMessage: null })
    render(<StateNotifier />)
    
    expect(screen.getByText('✅')).toBeInTheDocument()
    expect(screen.getByText('Ready for response')).toBeInTheDocument()
  })

  it('applies correct styling for different states', () => {
    useAppStore.setState({ state: 'waiting_user' })
    const { container } = render(<StateNotifier />)
    let notifier = container.firstChild as HTMLElement
    expect(notifier).toHaveClass('bg-green-50', 'border-green-200')

    useAppStore.setState({ state: 'server_error' })
    render(<StateNotifier />)
    notifier = container.firstChild as HTMLElement
    expect(notifier).toHaveClass('bg-red-50', 'border-red-200')
  })

  it('handles all state types', () => {
    const states: AppState[] = [
      'idle',
      'awaiting_data', 
      'waiting_user',
      'submitting',
      'server_error',
      'client_error'
    ]

    states.forEach(state => {
      useAppStore.setState({ state })
      const { container, unmount } = render(<StateNotifier />)
      expect(container.firstChild).not.toBeNull()
      unmount()
    })
  })
})