import type { Input } from '../types';
import { GridVisualization } from './GridVisualization';
import { MultiChannelGridVisualization } from './MultiChannelGridVisualization';
import { ScalarVisualization } from './ScalarVisualization';
import { Vector2DVisualization } from './Vector2DVisualization';
import { TimeSeriesVisualization } from './TimeSeriesVisualization';

interface Props {
  input: Input;
  className?: string;
}

export function VisualizationRenderer({ input, className = '' }: Props) {
  const viz = input.Visualization;
  
  if (viz.Grid) {
    return (
      <div className={className}>
        <GridVisualization input={input} />
      </div>
    );
  }
  
  if (viz.MultiGrid) {
    return (
      <div className={className}>
        <MultiChannelGridVisualization input={input} />
      </div>
    );
  }
  
  if (viz.Scalar) {
    return (
      <div className={className}>
        <ScalarVisualization input={input} />
      </div>
    );
  }
  
  if (viz.Vector) {
    return (
      <div className={className}>
        <Vector2DVisualization input={input} />
      </div>
    );
  }
  
  if (viz.TimeSeries) {
    return (
      <div className={className}>
        <TimeSeriesVisualization input={input} />
      </div>
    );
  }
  
  return (
    <div className={`${className} flex items-center justify-center text-gray-500`}>
      <div className="text-center">
        <div className="text-4xl mb-2">❓</div>
        <div>Unknown visualization type</div>
      </div>
    </div>
  );
}