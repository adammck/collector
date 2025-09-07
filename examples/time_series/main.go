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

	// Generate a sine wave with noise for sensor readings
	points := 20
	data := make([]float64, points)
	for i := 0; i < points; i++ {
		t := float64(i) / float64(points-1) * 4 * math.Pi
		baseValue := math.Sin(t) * 0.8
		noise := (rand.Float64() - 0.5) * 0.3
		data[i] = baseValue + noise
	}

	req := &pb.Request{
		Inputs: []*pb.Input{
			{
				Visualization: &pb.Input_TimeSeries{
					TimeSeries: &pb.TimeSeries{
						Label:    "Sensor Reading",
						Points:   int32(points),
						MinValue: -1.5,
						MaxValue: 1.5,
					},
				},
				Data: &pb.Data{
					Data: &pb.Data_Floats{
						Floats: &pb.Floats{Values: data},
					},
				},
			},
		},
		Output: &pb.OutputSchema{
			Output: &pb.OutputSchema_OptionList{
				OptionList: &pb.OptionListSchema{
					Options: []*pb.Option{
						{Label: "Pattern Normal", Hotkey: "n"},
						{Label: "Anomaly Detected", Hotkey: "a"},
						{Label: "Too Noisy", Hotkey: "t"},
						{Label: "Need More Data", Hotkey: "m"},
					},
				},
			},
		},
	}

	log.Printf("Sending time series with %d points", points)
	r, err := c.Collect(ctx, req)
	if err != nil {
		log.Fatalf("could not collect: %v", err)
	}
	log.Printf("Selected option index: %d", r.GetOutput().GetOptionList().Index)
}