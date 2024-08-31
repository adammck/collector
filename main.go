package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var input, output *string

func main() {
	port := flag.Int("port", 8000, "port to listen on")
	input = flag.String("input", "input/*.jsonl", "input glob (beware shell escaping)")
	output = flag.String("output", "examples.jsonl", "output file")
	flag.Parse()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/data.json", func(w http.ResponseWriter, r *http.Request) {
		contents, err := getInputFile(*input)

		if err == os.ErrNotExist {
			http.Error(w, "no input files found", http.StatusNotFound)
			return

		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data, err := makeResponse(contents)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	})

	http.HandleFunc("/submit", submitHandler)

	log.Printf("Input: %s\n", *input)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Listening on %s\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r.Body); err != nil {
		http.Error(w, fmt.Sprintf("io.Copy: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("%s\n", buf.String())

	f, err := os.OpenFile(*output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		http.Error(w, fmt.Sprintf("os.OpenFile: %v", err), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if _, err := buf.WriteTo(f); err != nil {
		http.Error(w, fmt.Sprintf("buf.WriteTo: %v", err), http.StatusInternalServerError)
		return
	}

	if _, err := f.WriteString("\n"); err != nil {
		http.Error(w, fmt.Sprintf("f.WriteString: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{}`))
}

func makeResponse(contents []byte) (*Data, error) {
	d := []int{}
	if err := json.Unmarshal(contents, &d); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}

	rows := 8
	cols := 8

	if len(d) != rows*cols {
		return nil, fmt.Errorf("wrong length: expected=%d, got=%d", rows*cols, len(d))
	}

	return &Data{
		Inputs: []Input{
			{
				UI: UI{
					Type: "grid2d",
					Grid2D: Grid2D{
						Rows: 8,
						Cols: 8,
					},
				},
				Data: d,
			},
		},
		Output: Output{
			Type: "onehot",
			OneHot: OneHot{
				Options: []OneHotOption{
					{Label: "forwards", Key: "ArrowUp"},
					{Label: "backwards", Key: "ArrowDown"},
					{Label: "left", Key: "ArrowLeft"},
					{Label: "right", Key: "ArrowRight"},
					{Label: "stop", Key: " "},
				},
			},
		},
	}, nil
}

type Data struct {
	Inputs []Input `json:"inputs"`
	Output Output  `json:"output"`
}

// input

type Grid2D struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
}

type UI struct {
	Type   string `json:"type"`
	Grid2D Grid2D `json:"grid2d,omitempty"`
}

type Input struct {
	UI   UI    `json:"ui"`
	Data []int `json:"data"`
}

// output

type OneHotOption struct {
	Label string `json:"label"`
	Key   string `json:"key"`
}

type OneHot struct {
	Options []OneHotOption `json:"options"`
}

type Output struct {
	Type   string `json:"type"`
	OneHot OneHot `json:"one_hot,omitempty"`
}

func getInputFile(pattern string) ([]byte, error) {
	for i := 0; i < 30; i++ {
		files, err := filepath.Glob(pattern)
		if err != nil {
			return []byte{}, err
		}

		if len(files) > 0 {
			// pick a random file
			i := rand.Intn(len(files))
			return consume(files[i])
		}

		time.Sleep(100 * time.Millisecond)
	}

	return []byte{}, os.ErrNotExist
}

func consume(fn string) ([]byte, error) {
	contents, err := os.ReadFile(fn)
	if err != nil {
		return []byte{}, fmt.Errorf("os.ReadFile: %w", err)
	}

	err = os.Remove(fn)
	if err != nil {
		return []byte{}, fmt.Errorf("os.Remove: %w", err)
	}

	return contents, nil
}
