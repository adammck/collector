import type { DataResponse, Proto, Input, Queue } from '../types'

export const mockQueue: Queue = {
  total: 5,
  active: 3,
  deferred: 2,
}

export const mockGridInput: Input = {
  Visualization: {
    Grid: {
      rows: 3,
      cols: 3,
    },
  },
  data: {
    Data: {
      Ints: {
        values: [1, 0, 1, 0, 1, 0, 1, 0, 1],
      },
    },
  },
}

export const mockScalarInput: Input = {
  Visualization: {
    Scalar: {
      label: 'temperature',
      min: 0,
      max: 100,
      unit: '°c',
    },
  },
  data: {
    Data: {
      Floats: {
        values: [72.5],
      },
    },
  },
}

export const mockVector2DInput: Input = {
  Visualization: {
    Vector: {
      label: 'velocity',
      maxMagnitude: 10,
    },
  },
  data: {
    Data: {
      Floats: {
        values: [3.2, 4.8],
      },
    },
  },
}

export const mockTimeSeriesInput: Input = {
  Visualization: {
    TimeSeries: {
      label: 'sensor data',
      points: 5,
      minValue: 0,
      maxValue: 10,
    },
  },
  data: {
    Data: {
      Floats: {
        values: [1.2, 3.4, 5.6, 7.8, 9.0],
      },
    },
  },
}

export const mockMultiChannelGridInput: Input = {
  Visualization: {
    MultiGrid: {
      rows: 2,
      cols: 2,
      channels: 3,
      channelNames: ['red', 'green', 'blue'],
    },
  },
  data: {
    Data: {
      Ints: {
        values: [255, 0, 0, 0, 255, 0, 0, 0, 255, 128, 128, 128],
      },
    },
  },
}

export const mockProto: Proto = {
  inputs: [mockGridInput],
  output: {
    Output: {
      OptionList: {
        options: [
          { label: 'option a', hotkey: 'a' },
          { label: 'option b', hotkey: 'b' },
          { label: 'option c', hotkey: 'c' },
        ],
      },
    },
  },
}

export const mockDataResponse: DataResponse = {
  uuid: 'test-uuid-123',
  proto: mockProto,
  queue: mockQueue,
}

export const mockMultiInputProto: Proto = {
  inputs: [mockGridInput, mockScalarInput, mockVector2DInput, mockTimeSeriesInput],
  output: {
    Output: {
      OptionList: {
        options: [
          { label: 'left', hotkey: 'a' },
          { label: 'right', hotkey: 'd' },
        ],
      },
    },
  },
}

export const mockAPIError = (status: number, message: string) => {
  const error = new Error(message)
  error.name = 'APIError'
  ;(error as any).status = status
  return error
}