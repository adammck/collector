import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '../test/utils'
import userEvent from '@testing-library/user-event'
import { OptionList } from './OptionList'
import type { Output } from '../types'

describe('OptionList', () => {
  const mockOutput: Output = {
    OptionList: {
      options: [
        { label: 'move left', hotkey: 'a' },
        { label: 'move right', hotkey: 'd' },
        { label: 'jump', hotkey: 'w' },
      ]
    }
  }

  const mockOnSubmit = vi.fn()

  beforeEach(() => {
    mockOnSubmit.mockClear()
  })

  it('renders all options correctly', () => {
    render(<OptionList output={mockOutput} onSubmit={mockOnSubmit} />)

    expect(screen.getByText('move left')).toBeInTheDocument()
    expect(screen.getByText('move right')).toBeInTheDocument()
    expect(screen.getByText('jump')).toBeInTheDocument()
  })

  it('displays hotkeys for each option', () => {
    render(<OptionList output={mockOutput} onSubmit={mockOnSubmit} />)

    expect(screen.getByText('a')).toBeInTheDocument()
    expect(screen.getByText('d')).toBeInTheDocument()
    expect(screen.getByText('w')).toBeInTheDocument()
  })

  it('calls onSubmit with correct index when option clicked', async () => {
    const user = userEvent.setup()
    render(<OptionList output={mockOutput} onSubmit={mockOnSubmit} />)

    await user.click(screen.getByText('move right'))
    expect(mockOnSubmit).toHaveBeenCalledWith(1)

    await user.click(screen.getByText('jump'))
    expect(mockOnSubmit).toHaveBeenCalledWith(2)
  })

  it('handles options without hotkeys', () => {
    const outputWithoutHotkeys: Output = {
      OptionList: {
        options: [
          { label: 'option 1' },
          { label: 'option 2' },
        ]
      }
    }

    render(<OptionList output={outputWithoutHotkeys} onSubmit={mockOnSubmit} />)

    expect(screen.getByText('option 1')).toBeInTheDocument()
    expect(screen.getByText('option 2')).toBeInTheDocument()
  })

  it('handles mixed options with and without hotkeys', () => {
    const mixedOutput: Output = {
      OptionList: {
        options: [
          { label: 'with hotkey', hotkey: 'x' },
          { label: 'without hotkey' },
        ]
      }
    }

    render(<OptionList output={mixedOutput} onSubmit={mockOnSubmit} />)

    expect(screen.getByText('with hotkey')).toBeInTheDocument()
    expect(screen.getByText('without hotkey')).toBeInTheDocument()
    expect(screen.getByText('x')).toBeInTheDocument()
  })

  it('has correct accessibility attributes', () => {
    render(<OptionList output={mockOutput} onSubmit={mockOnSubmit} />)

    const buttons = screen.getAllByRole('button')
    expect(buttons).toHaveLength(3)
    buttons.forEach(button => {
      expect(button.tagName).toBe('BUTTON')
    })
  })

  it('handles empty output', () => {
    const emptyOutput: Output = {}
    render(<OptionList output={emptyOutput} onSubmit={mockOnSubmit} />)
    
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('handles single option', async () => {
    const singleOutput: Output = {
      OptionList: {
        options: [{ label: 'only option', hotkey: 'o' }]
      }
    }
    const user = userEvent.setup()
    
    render(<OptionList output={singleOutput} onSubmit={mockOnSubmit} />)

    expect(screen.getByText('only option')).toBeInTheDocument()
    expect(screen.getByText('o')).toBeInTheDocument()

    await user.click(screen.getByText('only option'))
    expect(mockOnSubmit).toHaveBeenCalledWith(0)
  })

  it('renders correct structure and styling', () => {
    const { container } = render(<OptionList output={mockOutput} onSubmit={mockOnSubmit} />)
    
    const optionList = container.firstChild as HTMLElement
    expect(optionList).toHaveClass('space-y-3')

    const buttons = screen.getAllByRole('button')
    buttons.forEach(button => {
      expect(button).toHaveClass('w-full')
    })
  })

  it('handles disabled state', () => {
    const { container } = render(<OptionList output={mockOutput} disabled={true} onSubmit={mockOnSubmit} />)
    
    const buttons = screen.getAllByRole('button')
    buttons.forEach(button => {
      expect(button).toBeDisabled()
    })

    const optionList = container.firstChild as HTMLElement
    expect(optionList).toHaveClass('opacity-50')
  })

  it('handles keyboard shortcuts', async () => {
    render(<OptionList output={mockOutput} onSubmit={mockOnSubmit} />)

    // simulate keydown event
    const event = new KeyboardEvent('keydown', { key: 'a' })
    document.dispatchEvent(event)

    expect(mockOnSubmit).toHaveBeenCalledWith(0)
  })
})