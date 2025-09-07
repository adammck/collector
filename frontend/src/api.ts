import type { DataResponse, SubmitRequest, ErrorResponse } from './types';

export class APIError extends Error {
  status: number;
  code?: string;
  
  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = 'APIError';
    this.status = status;
    this.code = code;
  }
}

export async function fetchData(): Promise<DataResponse> {
  const response = await fetch('/data.json');
  
  if (response.status === 408) {
    throw new APIError('timeout', 408);
  }
  
  if (response.status === 503) {
    throw new APIError('service unavailable', 503);
  }
  
  if (!response.ok) {
    let errorMsg = `HTTP ${response.status}`;
    try {
      const error: ErrorResponse = await response.json();
      errorMsg = error.message || errorMsg;
    } catch {
      // ignore json parse errors
    }
    throw new APIError(errorMsg, response.status);
  }
  
  const contentType = response.headers.get('content-type');
  if (contentType !== 'application/json') {
    throw new APIError(`invalid content-type: ${contentType}`, response.status);
  }
  
  return response.json();
}

export async function submitResponse(uuid: string, index: number): Promise<void> {
  const data: SubmitRequest = {
    output: {
      optionList: {
        index,
      },
    },
  };
  
  const response = await fetch(`/submit/${uuid}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  
  if (!response.ok) {
    let errorMsg = `HTTP ${response.status}`;
    try {
      const error: ErrorResponse = await response.json();
      errorMsg = error.message || errorMsg;
    } catch {
      // ignore json parse errors
    }
    throw new APIError(errorMsg, response.status);
  }
}

export async function deferItem(uuid: string): Promise<DataResponse> {
  const response = await fetch(`/defer/${uuid}`, {
    method: 'POST',
  });
  
  if (!response.ok) {
    throw new APIError(`defer failed: ${response.status}`, response.status);
  }
  
  return response.json();
}