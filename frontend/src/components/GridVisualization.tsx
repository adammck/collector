import type { Input } from '../types';

interface Props {
  input: Input;
}

export function GridVisualization({ input }: Props) {
  const grid = input.Visualization.Grid;
  const values = input.data.Data.Ints?.values;
  
  if (!grid || !values) return null;
  
  const { rows, cols } = grid;
  
  return (
    <div className="flex items-center justify-center h-full">
      <div className="bg-gray-50 p-4 rounded-lg border-2 border-gray-200 shadow-inner">
        <table className="border-collapse font-mono text-lg">
          <tbody>
            {Array.from({ length: rows }, (_, r) => (
              <tr key={r}>
                {Array.from({ length: cols }, (_, c) => {
                  const value = values[r * cols + c];
                  return (
                    <td
                      key={c}
                      className={`border border-gray-300 text-center w-12 h-12 font-bold transition-colors ${
                        value 
                          ? 'bg-gradient-to-br from-gray-800 to-gray-900 text-white shadow-sm' 
                          : 'bg-white text-gray-300 hover:bg-gray-50'
                      }`}
                    >
                      {value || '0'}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
        <div className="text-center mt-3 text-sm text-gray-600">
          {rows}×{cols} grid
        </div>
      </div>
    </div>
  );
}