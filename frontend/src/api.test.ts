import { describe, it, expect, beforeEach, vi } from 'vitest'
import { fetchData, submitResponse, deferItem, APIError } from './api'
import { mockDataResponse } from './test/mocks'

// mock fetch globally
const mockFetch = vi.fn()
global.fetch = mockFetch

describe('APIError', () => {
  it('creates error with status and code', () => {
    const error = new APIError('test message', 404, 'NOT_FOUND')
    expect(error.message).toBe('test message')
    expect(error.status).toBe(404)
    expect(error.code).toBe('NOT_FOUND')
    expect(error.name).toBe('APIError')
  })
})

describe('fetchData', () => {
  beforeEach(() => {
    mockFetch.mockClear()
  })

  it('returns data on successful response', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      headers: {
        get: (name: string) => name === 'content-type' ? 'application/json' : null,
      },
      json: () => Promise.resolve(mockDataResponse),
    })

    const result = await fetchData()
    expect(result).toEqual(mockDataResponse)
    expect(mockFetch).toHaveBeenCalledWith('/data.json')
  })

  it('throws APIError on 408 timeout', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 408,
      headers: { get: () => null },
    })

    await expect(fetchData()).rejects.toThrow(new APIError('timeout', 408))
  })

  it('throws APIError on 503 service unavailable', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 503,
      headers: { get: () => null },
    })

    await expect(fetchData()).rejects.toThrow(new APIError('service unavailable', 503))
  })

  it('throws APIError with parsed error message on other failures', async () => {
    const errorResponse = { message: 'validation failed', code: 'INVALID_REQUEST' }
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 400,
      headers: { get: () => null },
      json: () => Promise.resolve(errorResponse),
    })

    await expect(fetchData()).rejects.toThrow(new APIError('validation failed', 400))
  })

  it('throws APIError with generic message when json parsing fails', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      headers: { get: () => null },
      json: () => Promise.reject(new Error('invalid json')),
    })

    await expect(fetchData()).rejects.toThrow(new APIError('HTTP 500', 500))
  })

  it('throws APIError on invalid content-type', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      headers: {
        get: (name: string) => name === 'content-type' ? 'text/html' : null,
      },
    })

    await expect(fetchData()).rejects.toThrow(new APIError('invalid content-type: text/html', 200))
  })
})

describe('submitResponse', () => {
  beforeEach(() => {
    mockFetch.mockClear()
  })

  it('submits response successfully', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
    })

    await expect(submitResponse('test-uuid', 1)).resolves.toBeUndefined()
    
    expect(mockFetch).toHaveBeenCalledWith('/submit/test-uuid', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        output: {
          optionList: {
            index: 1,
          },
        },
      }),
    })
  })

  it('throws APIError on failure', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 404,
      json: () => Promise.resolve({ message: 'not found' }),
    })

    await expect(submitResponse('invalid-uuid', 0)).rejects.toThrow(new APIError('not found', 404))
  })

  it('throws APIError with generic message when json parsing fails', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: () => Promise.reject(new Error('invalid json')),
    })

    await expect(submitResponse('test-uuid', 0)).rejects.toThrow(new APIError('HTTP 500', 500))
  })
})

describe('deferItem', () => {
  beforeEach(() => {
    mockFetch.mockClear()
  })

  it('defers item and returns new data', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve(mockDataResponse),
    })

    const result = await deferItem('test-uuid')
    expect(result).toEqual(mockDataResponse)
    
    expect(mockFetch).toHaveBeenCalledWith('/defer/test-uuid', {
      method: 'POST',
    })
  })

  it('throws APIError on failure', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 404,
    })

    await expect(deferItem('invalid-uuid')).rejects.toThrow(new APIError('defer failed: 404', 404))
  })
})