package main

import (
	"context"
	"flag"
	"log"
	"math"
	"math/rand"
	"time"

	pb "github.com/adammck/collector/proto/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "the address to connect to")
	flag.Parse()

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewCollectorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	// Create camera grid data (simulated depth map)
	gridData := make([]int64, 6*6)
	for i := 0; i < len(gridData); i++ {
		gridData[i] = rand.Int63n(10)
	}

	// Generate velocity vector
	angle := rand.Float64() * 2 * math.Pi
	magnitude := rand.Float64() * 5.0 + 1.0
	vx := magnitude * math.Cos(angle)
	vy := magnitude * math.Sin(angle)

	// Generate temperature
	temperature := 18.0 + rand.Float64()*12.0 // 18-30°C

	req := &pb.Request{
		Inputs: []*pb.Input{
			{
				// Depth camera
				Visualization: &pb.Input_Grid{
					Grid: &pb.Grid{
						Rows: 6,
						Cols: 6,
					},
				},
				Data: &pb.Data{
					Data: &pb.Data_Ints{
						Ints: &pb.Ints{Values: gridData},
					},
				},
			},
			{
				// Robot velocity
				Visualization: &pb.Input_Vector{
					Vector: &pb.Vector2D{
						Label:        "Robot Velocity",
						MaxMagnitude: 8.0,
					},
				},
				Data: &pb.Data{
					Data: &pb.Data_Floats{
						Floats: &pb.Floats{Values: []float64{vx, vy}},
					},
				},
			},
			{
				// Temperature sensor
				Visualization: &pb.Input_Scalar{
					Scalar: &pb.Scalar{
						Label: "Motor Temperature",
						Min:   15.0,
						Max:   35.0,
						Unit:  "°C",
					},
				},
				Data: &pb.Data{
					Data: &pb.Data_Floats{
						Floats: &pb.Floats{Values: []float64{temperature}},
					},
				},
			},
		},
		Output: &pb.OutputSchema{
			Output: &pb.OutputSchema_OptionList{
				OptionList: &pb.OptionListSchema{
					Options: []*pb.Option{
						{Label: "Continue Current Path", Hotkey: "c"},
						{Label: "Avoid Obstacle", Hotkey: "a"},
						{Label: "Emergency Stop", Hotkey: "s"},
						{Label: "Reduce Speed", Hotkey: "r"},
					},
				},
			},
		},
	}

	log.Printf("Sending multi-input request: depth grid + velocity (%.1f,%.1f) + temp %.1f°C", 
		vx, vy, temperature)
	r, err := c.Collect(ctx, req)
	if err != nil {
		log.Fatalf("could not collect: %v", err)
	}
	log.Printf("Selected option index: %d", r.GetOutput().GetOptionList().Index)
}