package main

import (
	"context"
	"flag"
	"log"
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

	// Create sample request with 8x8 grid and some integer data
	req := &pb.Request{
		Inputs: []*pb.Input{
			{
				Visualization: &pb.Input_Grid{
					Grid: &pb.Grid{
						Rows: 8,
						Cols: 8,
					},
				},
				Data: &pb.Data{
					Data: &pb.Data_Ints{
						Ints: &pb.Ints{
							Values: []int64{1, 2, 3, 4, 5, 6, 7, 8},
						},
					},
				},
			},
		},
		Output: &pb.OutputSchema{
			Output: &pb.OutputSchema_OptionList{
				OptionList: &pb.OptionListSchema{
					Options: []*pb.Option{
						{Label: "Option 1", Hotkey: "1"},
						{Label: "Option 2", Hotkey: "2"},
					},
				},
			},
		},
	}

	r, err := c.Collect(ctx, req)
	if err != nil {
		log.Fatalf("could not collect: %v", err)
	}
	log.Printf("Selected option index: %d", r.GetOutput().GetOptionList().Index)
}
