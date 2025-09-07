import { describe, it, expect, beforeEach } from 'vitest'
import { useAppStore } from './store'
import { mockQueue } from './test/mocks'
import type { AppState } from './types'

// helper to get fresh store state
const getStoreState = () => useAppStore.getState()

describe('useAppStore', () => {
  beforeEach(() => {
    // reset store to initial state
    useAppStore.setState({
      currentUUID: null,
      state: 'idle',
      stateMessage: null,
      queue: null,
    })
  })

  it('initializes with correct default state', () => {
    const state = getStoreState()
    expect(state.currentUUID).toBeNull()
    expect(state.state).toBe('idle')
    expect(state.stateMessage).toBeNull()
    expect(state.queue).toBeNull()
  })

  describe('setCurrentUUID', () => {
    it('sets UUID correctly', () => {
      const { setCurrentUUID } = getStoreState()
      setCurrentUUID('test-uuid-123')
      
      expect(getStoreState().currentUUID).toBe('test-uuid-123')
    })

    it('can clear UUID by setting to null', () => {
      const { setCurrentUUID } = getStoreState()
      setCurrentUUID('test-uuid-123')
      setCurrentUUID(null)
      
      expect(getStoreState().currentUUID).toBeNull()
    })
  })

  describe('setState', () => {
    it('sets state without message', () => {
      const { setState } = getStoreState()
      setState('waiting_user')
      
      const state = getStoreState()
      expect(state.state).toBe('waiting_user')
      expect(state.stateMessage).toBeNull()
    })

    it('sets state with message', () => {
      const { setState } = getStoreState()
      setState('server_error', 'connection failed')
      
      const state = getStoreState()
      expect(state.state).toBe('server_error')
      expect(state.stateMessage).toBe('connection failed')
    })

    it('clears previous message when setting state without message', () => {
      const { setState } = getStoreState()
      setState('server_error', 'connection failed')
      setState('waiting_user')
      
      const state = getStoreState()
      expect(state.state).toBe('waiting_user')
      expect(state.stateMessage).toBeNull()
    })

    it('handles all valid app states', () => {
      const { setState } = getStoreState()
      const validStates: AppState[] = [
        'idle',
        'awaiting_data', 
        'waiting_user',
        'submitting',
        'server_error',
        'client_error'
      ]

      validStates.forEach(state => {
        setState(state)
        expect(getStoreState().state).toBe(state)
      })
    })
  })

  describe('setQueue', () => {
    it('sets queue data correctly', () => {
      const { setQueue } = getStoreState()
      setQueue(mockQueue)
      
      expect(getStoreState().queue).toEqual(mockQueue)
    })

    it('can clear queue by setting to null', () => {
      const { setQueue } = getStoreState()
      setQueue(mockQueue)
      setQueue(null)
      
      expect(getStoreState().queue).toBeNull()
    })
  })

  describe('complex state transitions', () => {
    it('handles complete data flow', () => {
      const { setCurrentUUID, setState, setQueue } = getStoreState()
      
      // simulate receiving data
      setCurrentUUID('uuid-1')
      setQueue(mockQueue)
      setState('waiting_user')
      
      let state = getStoreState()
      expect(state.currentUUID).toBe('uuid-1')
      expect(state.queue).toEqual(mockQueue)
      expect(state.state).toBe('waiting_user')
      
      // simulate submitting
      setState('submitting')
      state = getStoreState()
      expect(state.state).toBe('submitting')
      
      // simulate error
      setState('client_error', 'network timeout')
      state = getStoreState()
      expect(state.state).toBe('client_error')
      expect(state.stateMessage).toBe('network timeout')
    })

    it('maintains uuid and queue through state changes', () => {
      const { setCurrentUUID, setState, setQueue } = getStoreState()
      
      setCurrentUUID('uuid-persistent')
      setQueue(mockQueue)
      
      // state changes shouldn't affect uuid/queue
      setState('waiting_user')
      setState('submitting')
      setState('server_error', 'timeout')
      
      const state = getStoreState()
      expect(state.currentUUID).toBe('uuid-persistent')
      expect(state.queue).toEqual(mockQueue)
    })
  })
})