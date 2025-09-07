import { describe, it, expect } from 'vitest'
import { render, screen } from '../test/utils'
import { GridVisualization } from './GridVisualization'
import type { Input } from '../types'

describe('GridVisualization', () => {
  const mockGridInput: Input = {
    Visualization: {
      Grid: {
        rows: 3,
        cols: 3,
      },
    },
    data: {
      Data: {
        Ints: {
          values: [1, 0, 1, 0, 1, 0, 1, 0, 1],
        },
      },
    },
  }

  it('renders grid with correct dimensions', () => {
    render(<GridVisualization input={mockGridInput} />)
    
    // should render 9 cells (3x3)
    const cells = document.querySelectorAll('td')
    expect(cells).toHaveLength(9)
  })

  it('displays values correctly', () => {
    render(<GridVisualization input={mockGridInput} />)
    
    expect(screen.getAllByText('1')).toHaveLength(5) // pattern has 5 ones
    expect(screen.getAllByText('0')).toHaveLength(4) // pattern has 4 zeros
  })

  it('applies different styling for non-zero values', () => {
    render(<GridVisualization input={mockGridInput} />)
    
    const cells = document.querySelectorAll('td')
    
    // first cell should have value 1 and dark styling
    expect(cells[0]).toHaveTextContent('1')
    expect(cells[0]).toHaveClass('bg-gradient-to-br', 'from-gray-800', 'to-gray-900')
    
    // second cell should have value 0 and light styling
    expect(cells[1]).toHaveTextContent('0')
    expect(cells[1]).toHaveClass('bg-white', 'text-gray-300')
  })

  it('handles missing grid data gracefully', () => {
    const invalidInput: Input = {
      Visualization: {},
      data: {
        Data: {
          Ints: {
            values: [],
          },
        },
      },
    }

    const { container } = render(<GridVisualization input={invalidInput} />)
    expect(container.firstChild).toBeNull()
  })

  it('handles missing data gracefully', () => {
    const invalidInput: Input = {
      Visualization: {
        Grid: {
          rows: 2,
          cols: 2,
        },
      },
      data: {
        Data: {},
      },
    }

    const { container } = render(<GridVisualization input={invalidInput} />)
    expect(container.firstChild).toBeNull()
  })

  it('renders correct table structure', () => {
    const { container } = render(<GridVisualization input={mockGridInput} />)
    
    expect(container.querySelector('table')).toBeInTheDocument()
    expect(container.querySelector('tbody')).toBeInTheDocument()
    
    const rows = container.querySelectorAll('tr')
    expect(rows).toHaveLength(3) // 3x3 grid has 3 rows
    
    rows.forEach(row => {
      const cells = row.querySelectorAll('td')
      expect(cells).toHaveLength(3) // each row has 3 cells
    })
  })

  it('handles different grid sizes', () => {
    const smallGridInput: Input = {
      Visualization: {
        Grid: {
          rows: 2,
          cols: 2,
        },
      },
      data: {
        Data: {
          Ints: {
            values: [1, 2, 3, 4],
          },
        },
      },
    }

    render(<GridVisualization input={smallGridInput} />)
    
    const cells = document.querySelectorAll('td')
    expect(cells).toHaveLength(4) // 2x2 grid
    
    expect(screen.getByText('1')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('4')).toBeInTheDocument()
  })

  it('has correct styling classes', () => {
    const { container } = render(<GridVisualization input={mockGridInput} />)
    
    const wrapper = container.firstChild as HTMLElement
    expect(wrapper).toHaveClass('flex', 'items-center', 'justify-center', 'h-full')
    
    const gridContainer = wrapper.firstChild as HTMLElement
    expect(gridContainer).toHaveClass('bg-gray-50', 'p-4', 'rounded-lg', 'border-2', 'border-gray-200', 'shadow-inner')
    
    const table = gridContainer.firstChild as HTMLElement
    expect(table).toHaveClass('border-collapse', 'font-mono', 'text-lg')
  })
})