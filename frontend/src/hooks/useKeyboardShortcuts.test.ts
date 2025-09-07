import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useKeyboardShortcuts } from './useKeyboardShortcuts'

describe('useKeyboardShortcuts', () => {
  const mockOnDefer = vi.fn()
  const mockOnNext = vi.fn()

  beforeEach(() => {
    mockOnDefer.mockClear()
    mockOnNext.mockClear()
  })

  afterEach(() => {
    // cleanup any event listeners
    document.removeEventListener('keydown', () => {})
  })

  it('calls onDefer when Ctrl+D is pressed', () => {
    renderHook(() => useKeyboardShortcuts({
      onDefer: mockOnDefer,
      onNext: mockOnNext,
    }))

    const event = new KeyboardEvent('keydown', {
      key: 'd',
      ctrlKey: true,
    })
    document.dispatchEvent(event)

    expect(mockOnDefer).toHaveBeenCalledTimes(1)
    expect(mockOnNext).not.toHaveBeenCalled()
  })

  it('calls onNext when Ctrl+N is pressed', () => {
    renderHook(() => useKeyboardShortcuts({
      onDefer: mockOnDefer,
      onNext: mockOnNext,
    }))

    const event = new KeyboardEvent('keydown', {
      key: 'n',
      ctrlKey: true,
    })
    document.dispatchEvent(event)

    expect(mockOnNext).toHaveBeenCalledTimes(1)
    expect(mockOnDefer).not.toHaveBeenCalled()
  })

  it('ignores keys without Ctrl modifier', () => {
    renderHook(() => useKeyboardShortcuts({
      onDefer: mockOnDefer,
      onNext: mockOnNext,
    }))

    // press 'd' without ctrl
    const dEvent = new KeyboardEvent('keydown', { key: 'd' })
    document.dispatchEvent(dEvent)

    // press 'n' without ctrl  
    const nEvent = new KeyboardEvent('keydown', { key: 'n' })
    document.dispatchEvent(nEvent)

    expect(mockOnDefer).not.toHaveBeenCalled()
    expect(mockOnNext).not.toHaveBeenCalled()
  })

  it('ignores Ctrl with other keys', () => {
    renderHook(() => useKeyboardShortcuts({
      onDefer: mockOnDefer,
      onNext: mockOnNext,
    }))

    const event = new KeyboardEvent('keydown', {
      key: 'x',
      ctrlKey: true,
    })
    document.dispatchEvent(event)

    expect(mockOnDefer).not.toHaveBeenCalled()
    expect(mockOnNext).not.toHaveBeenCalled()
  })

  it('is case-sensitive (only handles lowercase)', () => {
    renderHook(() => useKeyboardShortcuts({
      onDefer: mockOnDefer,
      onNext: mockOnNext,
    }))

    // uppercase D should not trigger
    const dEvent = new KeyboardEvent('keydown', {
      key: 'D',
      ctrlKey: true,
    })
    document.dispatchEvent(dEvent)

    // uppercase N should not trigger
    const nEvent = new KeyboardEvent('keydown', {
      key: 'N',
      ctrlKey: true,
    })
    document.dispatchEvent(nEvent)

    expect(mockOnDefer).not.toHaveBeenCalled()
    expect(mockOnNext).not.toHaveBeenCalled()
  })

  it('prevents default behavior for handled shortcuts', () => {
    renderHook(() => useKeyboardShortcuts({
      onDefer: mockOnDefer,
      onNext: mockOnNext,
    }))

    const event = new KeyboardEvent('keydown', {
      key: 'd',
      ctrlKey: true,
    })
    
    const preventDefaultSpy = vi.spyOn(event, 'preventDefault')
    document.dispatchEvent(event)

    expect(preventDefaultSpy).toHaveBeenCalled()
  })

  it('cleans up event listeners on unmount', () => {
    const removeEventListenerSpy = vi.spyOn(document, 'removeEventListener')
    
    const { unmount } = renderHook(() => useKeyboardShortcuts({
      onDefer: mockOnDefer,
      onNext: mockOnNext,
    }))

    unmount()

    expect(removeEventListenerSpy).toHaveBeenCalledWith('keydown', expect.any(Function))
  })

  it('re-registers listeners when handlers change', () => {
    const addEventListenerSpy = vi.spyOn(document, 'addEventListener')
    const removeEventListenerSpy = vi.spyOn(document, 'removeEventListener')

    const newOnDefer = vi.fn()
    let props = { onDefer: mockOnDefer, onNext: mockOnNext }

    const { rerender } = renderHook(() => useKeyboardShortcuts(props))

    const initialAddCount = addEventListenerSpy.mock.calls.length
    const initialRemoveCount = removeEventListenerSpy.mock.calls.length

    // update props - this should cause re-registration due to dependency change
    props = { onDefer: newOnDefer, onNext: mockOnNext }
    rerender()

    // should add/remove listeners again due to dependency change
    expect(addEventListenerSpy.mock.calls.length).toBeGreaterThan(initialAddCount)
    expect(removeEventListenerSpy.mock.calls.length).toBeGreaterThan(initialRemoveCount)

    // but should use new handler
    const event = new KeyboardEvent('keydown', {
      key: 'd',
      ctrlKey: true,
    })
    document.dispatchEvent(event)

    expect(newOnDefer).toHaveBeenCalledTimes(1)
    expect(mockOnDefer).not.toHaveBeenCalled()
  })
})