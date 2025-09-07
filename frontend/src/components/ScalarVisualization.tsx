import type { Input } from '../types';

interface Props {
  input: Input;
}

export function ScalarVisualization({ input }: Props) {
  const scalar = input.Visualization.Scalar;
  const value = input.data.Data.Floats?.values[0];
  
  if (!scalar || value === undefined) return null;
  
  const { label, min, max, unit } = scalar;
  const percent = ((value - min) / (max - min)) * 100;
  const clampedPercent = Math.max(0, Math.min(100, percent));
  
  // Color based on value position
  const getBarColor = (percentage: number) => {
    if (percentage < 33) return 'from-green-500 to-green-600';
    if (percentage < 66) return 'from-yellow-500 to-orange-500';
    return 'from-orange-500 to-red-500';
  };
  
  return (
    <div className="flex items-center justify-center h-full">
      <div className="bg-gray-50 p-6 rounded-lg border-2 border-gray-200 shadow-inner w-full max-w-sm">
        <div className="text-center mb-4">
          <h3 className="text-lg font-semibold text-gray-800">{label}</h3>
        </div>
        
        {/* Value display */}
        <div className="text-center mb-4">
          <div className="text-3xl font-bold text-gray-900">
            {value.toFixed(2)}
            {unit && <span className="text-lg text-gray-600 ml-1">{unit}</span>}
          </div>
        </div>
        
        {/* Progress bar */}
        <div className="relative">
          <div className="w-full h-6 bg-gray-200 rounded-full overflow-hidden">
            <div
              className={`h-full bg-gradient-to-r ${getBarColor(clampedPercent)} transition-all duration-300 ease-out`}
              style={{ width: `${clampedPercent}%` }}
            />
          </div>
          
          {/* Min/Max labels */}
          <div className="flex justify-between mt-2 text-sm text-gray-600">
            <span>{min}{unit && ` ${unit}`}</span>
            <span>{max}{unit && ` ${unit}`}</span>
          </div>
        </div>
        
        {/* Percentage indicator */}
        <div className="text-center mt-3 text-sm text-gray-500">
          {clampedPercent.toFixed(1)}% of range
        </div>
      </div>
    </div>
  );
}