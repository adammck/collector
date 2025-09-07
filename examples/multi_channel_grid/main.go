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

	// Create RGB image data (8x8x3 = 192 values)
	rgbData := make([]int64, 8*8*3)
	for r := 0; r < 8; r++ {
		for c := 0; c < 8; c++ {
			baseIndex := (r*8 + c) * 3
			
			// Create a simple pattern with different colors
			if (r+c)%2 == 0 {
				// Red squares
				rgbData[baseIndex] = 255   // R
				rgbData[baseIndex+1] = 100 // G
				rgbData[baseIndex+2] = 100 // B
			} else {
				// Blue squares
				rgbData[baseIndex] = 100   // R
				rgbData[baseIndex+1] = 100 // G
				rgbData[baseIndex+2] = 255 // B
			}
		}
	}

	req := &pb.Request{
		Inputs: []*pb.Input{
			{
				Visualization: &pb.Input_MultiGrid{
					MultiGrid: &pb.MultiChannelGrid{
						Rows:         8,
						Cols:         8,
						Channels:     3,
						ChannelNames: []string{"Red", "Green", "Blue"},
					},
				},
				Data: &pb.Data{
					Data: &pb.Data_Ints{
						Ints: &pb.Ints{Values: rgbData},
					},
				},
			},
		},
		Output: &pb.OutputSchema{
			Output: &pb.OutputSchema_OptionList{
				OptionList: &pb.OptionListSchema{
					Options: []*pb.Option{
						{Label: "Looks good", Hotkey: "g"},
						{Label: "Too red", Hotkey: "r"},
						{Label: "Too blue", Hotkey: "b"},
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