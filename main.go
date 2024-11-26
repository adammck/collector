package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	pb "github.com/adammck/collector/proto/gen"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

type pair struct {
	req *pb.Request
	res chan *pb.Response
}

var proxy chan *pair
var pending map[string]chan *pb.Response

func init() {
	proxy = make(chan *pair)
	pending = map[string]chan *pb.Response{}
}

func main() {
	hp := flag.Int("http-port", 8000, "port for http server to listen on")
	gp := flag.Int("grpc-port", 50051, "port for grpc server to listen on")
	flag.Parse()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/data.json", func(w http.ResponseWriter, r *http.Request) {

		// Block until the next request (from gRPC) is available.
		// TODO: Timeout
		p := <-proxy

		// Store the response chan for later.
		u := uuid.NewString()
		pending[u] = p.res

		err := validate(p.req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		b, err := protojson.Marshal(p.req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})

	http.HandleFunc("POST /submit/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		u := r.PathValue("uuid")
		if u == "" {
			http.Error(w, "missing: uuid", http.StatusNotFound)
			return
		}

		resCh, ok := pending[u]
		if !ok {
			http.Error(w, fmt.Sprintf("pending not found: %v", u), http.StatusNotFound)
			return
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res := &pb.Response{}
		err = protojson.Unmarshal(b, res)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Send the encoded response back to the gRPC server. It will send it
		// back to the caller, which is hopefully still waiting.
		resCh <- res
		close(resCh)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	})

	go func() {
		addr := fmt.Sprintf(":%d", *hp)
		log.Printf("Listening on %s\n", addr)
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	go func() {
		addr := fmt.Sprintf(":%d", *gp)
		log.Printf("Listening on %s\n", addr)

		lis, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		srv := grpc.NewServer()
		server := &collectorServer{}
		pb.RegisterCollectorServer(srv, server)

		if err := srv.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	select {}
}

func validate(req *pb.Request) error {
	return nil
}

type collectorServer struct {
	pb.UnsafeCollectorServer
}

func (s *collectorServer) Collect(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	resCh := make(chan *pb.Response, 1)

	proxy <- &pair{
		req: req,
		res: resCh,
	}

	// Wait for the response to arrive.
	// TODO(adammck): Timeout
	res := <-resCh

	return res, nil
}
