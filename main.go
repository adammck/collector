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
)

var output *string

func main() {
	port := flag.Int("port", 8000, "port to listen on")
	output = flag.String("output", "examples.jsonl", "output file")
	flag.Parse()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/data.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(getData())
	})

	http.HandleFunc("/submit", submitHandler)

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

func getData() Data {
	d := []int{
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	}

	d[rand.Intn(len(d))] = 1

	return Data{
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
	}
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
