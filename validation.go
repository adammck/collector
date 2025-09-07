package main

import (
	"fmt"
	"math"

	pb "github.com/adammck/collector/proto/gen"
)

func validate(req *pb.Request) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if len(req.Inputs) == 0 {
		return fmt.Errorf("request must have at least one input")
	}

	for i, input := range req.Inputs {
		if err := validateInput(input, i); err != nil {
			return fmt.Errorf("input %d: %w", i, err)
		}
	}

	if err := validateOutputSchema(req.Output); err != nil {
		return fmt.Errorf("output schema: %w", err)
	}

	return nil
}

func validateInput(input *pb.Input, index int) error {
	if input == nil {
		return fmt.Errorf("input cannot be nil")
	}

	switch v := input.Visualization.(type) {
	case *pb.Input_Grid:
		if err := validateGrid(v.Grid, input.Data); err != nil {
			return err
		}
	case *pb.Input_MultiGrid:
		if err := validateMultiChannelGrid(v.MultiGrid, input.Data); err != nil {
			return err
		}
	case *pb.Input_Scalar:
		if err := validateScalar(v.Scalar, input.Data); err != nil {
			return err
		}
	case *pb.Input_Vector:
		if err := validateVector2D(v.Vector, input.Data); err != nil {
			return err
		}
	case *pb.Input_TimeSeries:
		if err := validateTimeSeries(v.TimeSeries, input.Data); err != nil {
			return err
		}
	case nil:
		return fmt.Errorf("visualization is required")
	default:
		return fmt.Errorf("unsupported visualization type")
	}

	return validateData(input.Data)
}

func validateGrid(grid *pb.Grid, data *pb.Data) error {
	if grid == nil {
		return fmt.Errorf("grid cannot be nil")
	}

	if grid.Rows <= 0 || grid.Cols <= 0 {
		return fmt.Errorf("grid dimensions must be positive (got %dx%d)", grid.Rows, grid.Cols)
	}

	if grid.Rows > 100 || grid.Cols > 100 {
		return fmt.Errorf("grid too large (max 100x100, got %dx%d)", grid.Rows, grid.Cols)
	}

	if data == nil {
		return fmt.Errorf("data is required")
	}

	expectedSize := int(grid.Rows * grid.Cols)

	switch d := data.Data.(type) {
	case *pb.Data_Ints:
		if d.Ints == nil {
			return fmt.Errorf("ints data cannot be nil")
		}
		if len(d.Ints.Values) != expectedSize {
			return fmt.Errorf("data size %d doesn't match grid size %d", len(d.Ints.Values), expectedSize)
		}
	case *pb.Data_Floats:
		if d.Floats == nil {
			return fmt.Errorf("floats data cannot be nil")
		}
		if len(d.Floats.Values) != expectedSize {
			return fmt.Errorf("data size %d doesn't match grid size %d", len(d.Floats.Values), expectedSize)
		}
	case nil:
		return fmt.Errorf("data type is required")
	default:
		return fmt.Errorf("unsupported data type")
	}

	return nil
}

func validateMultiChannelGrid(grid *pb.MultiChannelGrid, data *pb.Data) error {
	if grid == nil {
		return fmt.Errorf("multi-channel grid cannot be nil")
	}

	if grid.Rows <= 0 || grid.Cols <= 0 {
		return fmt.Errorf("grid dimensions must be positive (got %dx%d)", grid.Rows, grid.Cols)
	}

	if grid.Rows > 100 || grid.Cols > 100 {
		return fmt.Errorf("grid too large (max 100x100, got %dx%d)", grid.Rows, grid.Cols)
	}

	if grid.Channels <= 0 {
		return fmt.Errorf("channel count must be positive (got %d)", grid.Channels)
	}

	if grid.Channels > 10 {
		return fmt.Errorf("too many channels (max 10, got %d)", grid.Channels)
	}

	if len(grid.ChannelNames) > 0 && len(grid.ChannelNames) != int(grid.Channels) {
		return fmt.Errorf("channel names count %d doesn't match channel count %d", 
			len(grid.ChannelNames), grid.Channels)
	}

	if data == nil {
		return fmt.Errorf("data is required")
	}

	expectedSize := int(grid.Rows * grid.Cols * grid.Channels)

	switch d := data.Data.(type) {
	case *pb.Data_Ints:
		if d.Ints == nil {
			return fmt.Errorf("ints data cannot be nil")
		}
		if len(d.Ints.Values) != expectedSize {
			return fmt.Errorf("data size %d doesn't match expected size %d (rows*cols*channels=%d*%d*%d)", 
				len(d.Ints.Values), expectedSize, grid.Rows, grid.Cols, grid.Channels)
		}
	case *pb.Data_Floats:
		if d.Floats == nil {
			return fmt.Errorf("floats data cannot be nil")
		}
		if len(d.Floats.Values) != expectedSize {
			return fmt.Errorf("data size %d doesn't match expected size %d (rows*cols*channels=%d*%d*%d)", 
				len(d.Floats.Values), expectedSize, grid.Rows, grid.Cols, grid.Channels)
		}
	case nil:
		return fmt.Errorf("data type is required")
	default:
		return fmt.Errorf("unsupported data type")
	}

	return nil
}

func validateScalar(scalar *pb.Scalar, data *pb.Data) error {
	if scalar == nil {
		return fmt.Errorf("scalar cannot be nil")
	}

	if scalar.Label == "" {
		return fmt.Errorf("scalar label is required")
	}

	if scalar.Min >= scalar.Max {
		return fmt.Errorf("scalar min %f must be less than max %f", scalar.Min, scalar.Max)
	}

	if data == nil {
		return fmt.Errorf("data is required")
	}

	switch d := data.Data.(type) {
	case *pb.Data_Floats:
		if d.Floats == nil {
			return fmt.Errorf("floats data cannot be nil")
		}
		if len(d.Floats.Values) != 1 {
			return fmt.Errorf("scalar requires exactly 1 float value (got %d)", len(d.Floats.Values))
		}
		value := d.Floats.Values[0]
		if value < scalar.Min || value > scalar.Max {
			return fmt.Errorf("scalar value %f is outside range [%f, %f]", value, scalar.Min, scalar.Max)
		}
	default:
		return fmt.Errorf("scalar visualization requires float data")
	}

	return nil
}

func validateVector2D(vector *pb.Vector2D, data *pb.Data) error {
	if vector == nil {
		return fmt.Errorf("vector cannot be nil")
	}

	if vector.Label == "" {
		return fmt.Errorf("vector label is required")
	}

	if vector.MaxMagnitude <= 0 {
		return fmt.Errorf("vector max_magnitude must be positive (got %f)", vector.MaxMagnitude)
	}

	if data == nil {
		return fmt.Errorf("data is required")
	}

	switch d := data.Data.(type) {
	case *pb.Data_Floats:
		if d.Floats == nil {
			return fmt.Errorf("floats data cannot be nil")
		}
		if len(d.Floats.Values) != 2 {
			return fmt.Errorf("vector requires exactly 2 float values (got %d)", len(d.Floats.Values))
		}
		x, y := d.Floats.Values[0], d.Floats.Values[1]
		magnitude := math.Sqrt(x*x + y*y)
		if magnitude > vector.MaxMagnitude {
			return fmt.Errorf("vector magnitude %f exceeds max_magnitude %f", magnitude, vector.MaxMagnitude)
		}
	default:
		return fmt.Errorf("vector visualization requires float data")
	}

	return nil
}

func validateTimeSeries(timeSeries *pb.TimeSeries, data *pb.Data) error {
	if timeSeries == nil {
		return fmt.Errorf("time series cannot be nil")
	}

	if timeSeries.Label == "" {
		return fmt.Errorf("time series label is required")
	}

	if timeSeries.Points <= 0 {
		return fmt.Errorf("time series points must be positive (got %d)", timeSeries.Points)
	}

	if timeSeries.Points > 1000 {
		return fmt.Errorf("time series has too many points (max 1000, got %d)", timeSeries.Points)
	}

	if timeSeries.MinValue >= timeSeries.MaxValue {
		return fmt.Errorf("time series min_value %f must be less than max_value %f", 
			timeSeries.MinValue, timeSeries.MaxValue)
	}

	if data == nil {
		return fmt.Errorf("data is required")
	}

	switch d := data.Data.(type) {
	case *pb.Data_Floats:
		if d.Floats == nil {
			return fmt.Errorf("floats data cannot be nil")
		}
		expectedSize := int(timeSeries.Points)
		if len(d.Floats.Values) != expectedSize {
			return fmt.Errorf("data size %d doesn't match expected points %d", 
				len(d.Floats.Values), expectedSize)
		}
		for i, v := range d.Floats.Values {
			if v < timeSeries.MinValue || v > timeSeries.MaxValue {
				return fmt.Errorf("time series value at index %d (%f) is outside range [%f, %f]", 
					i, v, timeSeries.MinValue, timeSeries.MaxValue)
			}
		}
	default:
		return fmt.Errorf("time series visualization requires float data")
	}

	return nil
}

func validateData(data *pb.Data) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	switch d := data.Data.(type) {
	case *pb.Data_Ints:
		return nil
	case *pb.Data_Floats:
		if d.Floats == nil {
			return fmt.Errorf("floats data cannot be nil")
		}
		for i, v := range d.Floats.Values {
			if math.IsNaN(v) {
				return fmt.Errorf("float value at index %d is NaN", i)
			}
			if math.IsInf(v, 0) {
				return fmt.Errorf("float value at index %d is infinite", i)
			}
		}
		return nil
	case nil:
		return fmt.Errorf("data type is required")
	default:
		return fmt.Errorf("unsupported data type")
	}
}

func validateOutputSchema(schema *pb.OutputSchema) error {
	if schema == nil {
		return fmt.Errorf("output schema is required")
	}

	switch s := schema.Output.(type) {
	case *pb.OutputSchema_OptionList:
		if s.OptionList == nil {
			return fmt.Errorf("option list cannot be nil")
		}
		if len(s.OptionList.Options) < 2 {
			return fmt.Errorf("option list must have at least 2 options (got %d)", len(s.OptionList.Options))
		}

		hotkeys := make(map[string]bool)
		for i, opt := range s.OptionList.Options {
			if opt == nil {
				return fmt.Errorf("option %d cannot be nil", i)
			}
			if opt.Label == "" {
				return fmt.Errorf("option %d label cannot be empty", i)
			}
			if len(opt.Hotkey) != 1 {
				return fmt.Errorf("option %d hotkey must be single character (got %q)", i, opt.Hotkey)
			}
			if hotkeys[opt.Hotkey] {
				return fmt.Errorf("duplicate hotkey %q found at option %d", opt.Hotkey, i)
			}
			hotkeys[opt.Hotkey] = true
		}
		return nil
	case nil:
		return fmt.Errorf("output type is required")
	default:
		return fmt.Errorf("unsupported output schema type")
	}
}