import type { Input } from '../types';
import { VisualizationRenderer } from './VisualizationRenderer';

interface Props {
  inputs: Input[];
}

export function VisualizationLayout({ inputs }: Props) {
  if (inputs.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-gray-500">
        <div className="text-center">
          <div className="text-4xl mb-2">📊</div>
          <div>No data available</div>
        </div>
      </div>
    );
  }
  
  if (inputs.length === 1) {
    return (
      <VisualizationRenderer 
        input={inputs[0]} 
        className="w-full h-full"
      />
    );
  }
  
  if (inputs.length === 2) {
    return (
      <div className="flex w-full h-full gap-4">
        <VisualizationRenderer 
          input={inputs[0]} 
          className="flex-1 bg-white rounded-lg border border-gray-200 shadow-sm overflow-hidden"
        />
        <VisualizationRenderer 
          input={inputs[1]} 
          className="flex-1 bg-white rounded-lg border border-gray-200 shadow-sm overflow-hidden"
        />
      </div>
    );
  }
  
  if (inputs.length <= 4) {
    return (
      <div className="grid grid-cols-2 gap-4 w-full h-full">
        {inputs.map((input, index) => (
          <VisualizationRenderer 
            key={index}
            input={input} 
            className="bg-white rounded-lg border border-gray-200 shadow-sm overflow-hidden"
          />
        ))}
      </div>
    );
  }
  
  // For more than 4 inputs, create a dynamic grid
  const cols = Math.ceil(Math.sqrt(inputs.length));
  const gridStyle = {
    gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))`,
  };
  
  return (
    <div className="grid gap-3 w-full h-full" style={gridStyle}>
      {inputs.map((input, index) => (
        <VisualizationRenderer 
          key={index}
          input={input} 
          className="bg-white rounded-lg border border-gray-200 shadow-sm overflow-hidden min-h-0"
        />
      ))}
    </div>
  );
}