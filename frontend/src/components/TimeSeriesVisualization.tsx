import { useEffect, useRef } from 'react';
import type { Input } from '../types';

interface Props {
  input: Input;
}

export function TimeSeriesVisualization({ input }: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const timeSeries = input.Visualization.TimeSeries;
  const values = input.data.Data.Floats?.values;
  
  useEffect(() => {
    if (!timeSeries || !values || !canvasRef.current) return;
    
    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    
    const { points, minValue, maxValue } = timeSeries;
    
    // Set canvas size and clear
    canvas.width = 400;
    canvas.height = 200;
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    
    const padding = 40;
    const chartWidth = canvas.width - 2 * padding;
    const chartHeight = canvas.height - 2 * padding;
    const valueRange = maxValue - minValue;
    
    // Draw background
    ctx.fillStyle = '#f9fafb';
    ctx.fillRect(padding, padding, chartWidth, chartHeight);
    
    // Draw grid lines
    ctx.strokeStyle = '#e5e7eb';
    ctx.lineWidth = 1;
    ctx.setLineDash([2, 2]);
    
    // Horizontal grid lines (value levels)
    const numHorizontalLines = 5;
    for (let i = 0; i <= numHorizontalLines; i++) {
      const y = padding + (chartHeight * i) / numHorizontalLines;
      ctx.beginPath();
      ctx.moveTo(padding, y);
      ctx.lineTo(padding + chartWidth, y);
      ctx.stroke();
    }
    
    // Vertical grid lines (time points)
    const numVerticalLines = Math.min(10, points - 1);
    for (let i = 0; i <= numVerticalLines; i++) {
      const x = padding + (chartWidth * i) / numVerticalLines;
      ctx.beginPath();
      ctx.moveTo(x, padding);
      ctx.lineTo(x, padding + chartHeight);
      ctx.stroke();
    }
    
    ctx.setLineDash([]);
    
    // Draw axes
    ctx.strokeStyle = '#374151';
    ctx.lineWidth = 2;
    ctx.beginPath();
    // Y axis
    ctx.moveTo(padding, padding);
    ctx.lineTo(padding, padding + chartHeight);
    // X axis
    ctx.moveTo(padding, padding + chartHeight);
    ctx.lineTo(padding + chartWidth, padding + chartHeight);
    ctx.stroke();
    
    // Draw data line
    if (values.length >= 2) {
      ctx.strokeStyle = '#3b82f6';
      ctx.lineWidth = 2;
      ctx.beginPath();
      
      for (let i = 0; i < values.length; i++) {
        const x = padding + (chartWidth * i) / (values.length - 1);
        const normalizedValue = (values[i] - minValue) / valueRange;
        const y = padding + chartHeight * (1 - normalizedValue); // Flip Y axis
        
        if (i === 0) {
          ctx.moveTo(x, y);
        } else {
          ctx.lineTo(x, y);
        }
      }
      
      ctx.stroke();
      
      // Draw data points
      ctx.fillStyle = '#3b82f6';
      for (let i = 0; i < values.length; i++) {
        const x = padding + (chartWidth * i) / (values.length - 1);
        const normalizedValue = (values[i] - minValue) / valueRange;
        const y = padding + chartHeight * (1 - normalizedValue);
        
        ctx.beginPath();
        ctx.arc(x, y, 3, 0, 2 * Math.PI);
        ctx.fill();
      }
    }
    
    // Draw axis labels
    ctx.fillStyle = '#6b7280';
    ctx.font = '12px Inter, sans-serif';
    ctx.textAlign = 'right';
    ctx.textBaseline = 'middle';
    
    // Y axis labels (values)
    for (let i = 0; i <= numHorizontalLines; i++) {
      const value = maxValue - (valueRange * i) / numHorizontalLines;
      const y = padding + (chartHeight * i) / numHorizontalLines;
      ctx.fillText(value.toFixed(1), padding - 5, y);
    }
    
    // X axis labels (time points)
    ctx.textAlign = 'center';
    ctx.textBaseline = 'top';
    for (let i = 0; i <= Math.min(5, values.length - 1); i++) {
      const timePoint = Math.round((i * (values.length - 1)) / 5);
      const x = padding + (chartWidth * timePoint) / (values.length - 1);
      ctx.fillText(timePoint.toString(), x, padding + chartHeight + 5);
    }
    
  }, [timeSeries, values]);
  
  if (!timeSeries || !values) return null;
  
  const { label, points, minValue, maxValue } = timeSeries;
  const currentValue = values[values.length - 1];
  const minVal = Math.min(...values);
  const maxVal = Math.max(...values);
  const avgVal = values.reduce((sum, val) => sum + val, 0) / values.length;
  
  return (
    <div className="flex items-center justify-center h-full">
      <div className="bg-gray-50 p-4 rounded-lg border-2 border-gray-200 shadow-inner">
        <div className="text-center mb-3">
          <h3 className="text-lg font-semibold text-gray-800">{label}</h3>
        </div>
        
        <canvas
          ref={canvasRef}
          className="border border-gray-300 rounded bg-white"
        />
        
        <div className="mt-3 grid grid-cols-2 gap-4 text-sm text-gray-600">
          <div className="space-y-1">
            <div><span className="font-medium">Current:</span> {currentValue?.toFixed(2)}</div>
            <div><span className="font-medium">Average:</span> {avgVal.toFixed(2)}</div>
          </div>
          <div className="space-y-1">
            <div><span className="font-medium">Min:</span> {minVal.toFixed(2)}</div>
            <div><span className="font-medium">Max:</span> {maxVal.toFixed(2)}</div>
          </div>
        </div>
        
        <div className="text-center mt-2 text-xs text-gray-500">
          {values.length} of {points} points • Range: [{minValue}, {maxValue}]
        </div>
      </div>
    </div>
  );
}