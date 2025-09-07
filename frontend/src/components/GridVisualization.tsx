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
    <div className="flex items-center justify-center h-full p-2">
      <table className="border-collapse font-mono text-xl">
        <tbody>
          {Array.from({ length: rows }, (_, r) => (
            <tr key={r}>
              {Array.from({ length: cols }, (_, c) => {
                const value = values[r * cols + c];
                return (
                  <td
                    key={c}
                    className={`border border-gray-300 p-2 text-center w-10 h-10 ${
                      value ? 'bg-black text-white' : 'bg-white text-gray-200'
                    }`}
                  >
                    {value}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}