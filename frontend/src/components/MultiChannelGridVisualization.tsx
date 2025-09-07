import { useEffect, useRef } from 'react';
import type { Input } from '../types';

interface Props {
  input: Input;
}

export function MultiChannelGridVisualization({ input }: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const multiGrid = input.Visualization.MultiGrid;
  const values = input.data.Data.Ints?.values || input.data.Data.Floats?.values;
  
  useEffect(() => {
    if (!multiGrid || !values || !canvasRef.current) return;
    
    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    
    const { rows, cols, channels } = multiGrid;
    
    // Set canvas size
    canvas.width = cols;
    canvas.height = rows;
    
    // Create image data
    const imageData = ctx.createImageData(cols, rows);
    const data = imageData.data;
    
    // Fill pixel data
    for (let r = 0; r < rows; r++) {
      for (let c = 0; c < cols; c++) {
        const pixelIndex = (r * cols + c) * 4; // RGBA
        const dataIndex = (r * cols + c) * channels;
        
        if (channels === 1) {
          // Grayscale
          const value = Math.floor(values[dataIndex]);
          data[pixelIndex] = value;     // R
          data[pixelIndex + 1] = value; // G  
          data[pixelIndex + 2] = value; // B
          data[pixelIndex + 3] = 255;   // A
        } else if (channels === 3) {
          // RGB
          data[pixelIndex] = Math.floor(values[dataIndex]);     // R
          data[pixelIndex + 1] = Math.floor(values[dataIndex + 1]); // G
          data[pixelIndex + 2] = Math.floor(values[dataIndex + 2]); // B
          data[pixelIndex + 3] = 255;                           // A
        } else if (channels === 4) {
          // RGBA
          data[pixelIndex] = Math.floor(values[dataIndex]);     // R
          data[pixelIndex + 1] = Math.floor(values[dataIndex + 1]); // G
          data[pixelIndex + 2] = Math.floor(values[dataIndex + 2]); // B
          data[pixelIndex + 3] = Math.floor(values[dataIndex + 3]); // A
        } else {
          // Multi-channel: use first 3 channels as RGB
          data[pixelIndex] = Math.floor(values[dataIndex] || 0);
          data[pixelIndex + 1] = Math.floor(values[dataIndex + 1] || 0);
          data[pixelIndex + 2] = Math.floor(values[dataIndex + 2] || 0);
          data[pixelIndex + 3] = 255;
        }
      }
    }
    
    ctx.putImageData(imageData, 0, 0);
  }, [multiGrid, values]);
  
  if (!multiGrid || !values) return null;
  
  const { rows, cols, channels, channelNames } = multiGrid;
  
  return (
    <div className="flex items-center justify-center h-full">
      <div className="bg-gray-50 p-4 rounded-lg border-2 border-gray-200 shadow-inner">
        <canvas
          ref={canvasRef}
          className="border border-gray-300 max-w-full max-h-96"
          style={{
            imageRendering: 'pixelated',
            width: `${Math.min(400, cols * 4)}px`,
            height: `${Math.min(400, rows * 4)}px`,
          }}
        />
        <div className="text-center mt-3 text-sm text-gray-600">
          <div>{rows}×{cols} grid ({channels} channels)</div>
          {channelNames.length > 0 && (
            <div className="mt-1 text-xs text-gray-500">
              {channelNames.join(', ')}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}