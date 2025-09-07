export interface GridVisualization {
  rows: number;
  cols: number;
}

export interface Visualization {
  Grid?: GridVisualization;
}

export interface IntData {
  values: number[];
}

export interface Data {
  Ints?: IntData;
}

export interface InputData {
  Data: Data;
}

export interface Input {
  Visualization: Visualization;
  data: InputData;
}

export interface Option {
  label: string;
  hotkey?: string;
}

export interface OptionListOutput {
  options: Option[];
}

export interface Output {
  OptionList?: OptionListOutput;
}

export interface Proto {
  inputs?: Input[];
  output?: {
    Output: Output;
  };
}

export interface Queue {
  total: number;
  active: number;
  deferred: number;
}

export interface DataResponse {
  uuid: string;
  proto: Proto;
  queue: Queue;
}

export interface SubmitRequest {
  output: {
    optionList: {
      index: number;
    };
  };
}

export interface ErrorResponse {
  code?: string;
  message: string;
  details?: string;
}

export type AppState = 'idle' | 'awaiting_data' | 'waiting_user' | 'submitting' | 'server_error' | 'client_error';