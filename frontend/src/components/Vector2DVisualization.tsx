import { useEffect, useRef } from 'react';
import type { Input } from '../types';

interface Props {
  input: Input;
}

export function Vector2DVisualization({ input }: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const vector = input.Visualization.Vector;
  const values = input.data.Data.Floats?.values;
  
  useEffect(() => {
    if (!vector || !values || values.length !== 2 || !canvasRef.current) return;
    
    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    
    const [x, y] = values;
    const { maxMagnitude } = vector;
    
    // Set canvas size and clear
    canvas.width = 300;
    canvas.height = 300;
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    
    const centerX = canvas.width / 2;
    const centerY = canvas.height / 2;
    const maxRadius = Math.min(canvas.width, canvas.height) / 2 - 20;
    
    // Draw coordinate system
    ctx.strokeStyle = '#cbd5e1';
    ctx.lineWidth = 1;
    ctx.setLineDash([5, 5]);
    
    // X and Y axes
    ctx.beginPath();
    ctx.moveTo(0, centerY);
    ctx.lineTo(canvas.width, centerY);
    ctx.moveTo(centerX, 0);
    ctx.lineTo(centerX, canvas.height);
    ctx.stroke();
    
    // Draw circular magnitude guides
    const numCircles = 3;
    for (let i = 1; i <= numCircles; i++) {
      const radius = (maxRadius * i) / numCircles;
      ctx.beginPath();
      ctx.arc(centerX, centerY, radius, 0, 2 * Math.PI);
      ctx.stroke();
    }
    
    ctx.setLineDash([]);
    
    // Calculate vector position
    const scale = maxRadius / maxMagnitude;
    const endX = centerX + x * scale;
    const endY = centerY - y * scale; // Flip Y for screen coordinates
    
    // Draw vector
    const magnitude = Math.sqrt(x * x + y * y);
    const normalizedMagnitude = magnitude / maxMagnitude;
    
    // Color based on magnitude
    let strokeColor: string;
    if (normalizedMagnitude < 0.33) {
      strokeColor = '#10b981'; // green
    } else if (normalizedMagnitude < 0.66) {
      strokeColor = '#f59e0b'; // orange
    } else {
      strokeColor = '#ef4444'; // red
    }
    
    ctx.strokeStyle = strokeColor;
    ctx.fillStyle = strokeColor;
    ctx.lineWidth = 3;
    
    // Draw vector line
    ctx.beginPath();
    ctx.moveTo(centerX, centerY);
    ctx.lineTo(endX, endY);
    ctx.stroke();
    
    // Draw arrowhead
    if (magnitude > 0.01) {
      const angle = Math.atan2(-y, x); // Negative y for screen coordinates
      const arrowLength = 12;
      const arrowAngle = Math.PI / 6;
      
      ctx.save();
      ctx.translate(endX, endY);
      ctx.rotate(angle);
      
      ctx.beginPath();
      ctx.moveTo(0, 0);
      ctx.lineTo(-arrowLength, -arrowLength * Math.tan(arrowAngle));
      ctx.lineTo(-arrowLength, arrowLength * Math.tan(arrowAngle));
      ctx.closePath();
      ctx.fill();
      
      ctx.restore();
    }
    
    // Draw center point
    ctx.fillStyle = '#374151';
    ctx.beginPath();
    ctx.arc(centerX, centerY, 4, 0, 2 * Math.PI);
    ctx.fill();
    
  }, [vector, values]);
  
  if (!vector || !values || values.length !== 2) return null;
  
  const [x, y] = values;
  const { label, maxMagnitude } = vector;
  const magnitude = Math.sqrt(x * x + y * y);
  const angle = Math.atan2(y, x) * (180 / Math.PI);
  
  return (
    <div className="flex items-center justify-center h-full">
      <div className="bg-gray-50 p-4 rounded-lg border-2 border-gray-200 shadow-inner">
        <div className="text-center mb-3">
          <h3 className="text-lg font-semibold text-gray-800">{label}</h3>
        </div>
        
        <canvas
          ref={canvasRef}
          className="border border-gray-300 rounded"
        />
        
        <div className="mt-3 text-sm text-gray-600 space-y-1">
          <div className="grid grid-cols-2 gap-4">
            <div className="text-center">
              <div className="font-semibold">X: {x.toFixed(2)}</div>
              <div className="font-semibold">Y: {y.toFixed(2)}</div>
            </div>
            <div className="text-center">
              <div>Magnitude: {magnitude.toFixed(2)}</div>
              <div>Angle: {angle.toFixed(1)}°</div>
            </div>
          </div>
          <div className="text-center text-xs text-gray-500 mt-2">
            Max magnitude: {maxMagnitude}
          </div>
        </div>
      </div>
    </div>
  );
}