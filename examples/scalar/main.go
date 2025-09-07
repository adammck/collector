package main

import (
	"context"
	"flag"
	"log"
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

	// Generate a random temperature reading
	temperature := 15.0 + rand.Float64()*25.0 // 15-40°C

	req := &pb.Request{
		Inputs: []*pb.Input{
			{
				Visualization: &pb.Input_Scalar{
					Scalar: &pb.Scalar{
						Label: "Temperature",
						Min:   0.0,
						Max:   50.0,
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
						{Label: "Too Cold", Hotkey: "c"},
						{Label: "Just Right", Hotkey: "r"},
						{Label: "Too Hot", Hotkey: "h"},
					},
				},
			},
		},
	}

	log.Printf("Sending temperature reading: %.1f°C", temperature)
	r, err := c.Collect(ctx, req)
	if err != nil {
		log.Fatalf("could not collect: %v", err)
	}
	log.Printf("Selected option index: %d", r.GetOutput().GetOptionList().Index)
}