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

	// Generate a random velocity vector
	angle := rand.Float64() * 2 * math.Pi
	magnitude := rand.Float64() * 8.0 + 1.0 // 1-9 m/s
	vx := magnitude * math.Cos(angle)
	vy := magnitude * math.Sin(angle)

	req := &pb.Request{
		Inputs: []*pb.Input{
			{
				Visualization: &pb.Input_Vector{
					Vector: &pb.Vector2D{
						Label:        "Velocity",
						MaxMagnitude: 10.0,
					},
				},
				Data: &pb.Data{
					Data: &pb.Data_Floats{
						Floats: &pb.Floats{Values: []float64{vx, vy}},
					},
				},
			},
		},
		Output: &pb.OutputSchema{
			Output: &pb.OutputSchema_OptionList{
				OptionList: &pb.OptionListSchema{
					Options: []*pb.Option{
						{Label: "Move Forward", Hotkey: "f"},
						{Label: "Turn Left", Hotkey: "l"},
						{Label: "Turn Right", Hotkey: "r"},
						{Label: "Stop", Hotkey: "s"},
					},
				},
			},
		},
	}

	log.Printf("Sending velocity vector: (%.2f, %.2f) m/s, magnitude: %.2f m/s", vx, vy, magnitude)
	r, err := c.Collect(ctx, req)
	if err != nil {
		log.Fatalf("could not collect: %v", err)
	}
	log.Printf("Selected option index: %d", r.GetOutput().GetOptionList().Index)
}