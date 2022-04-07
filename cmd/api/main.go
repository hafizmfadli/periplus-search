package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/go-chi/chi/v5"
	"github.com/hafizmfadli/periplus-search/pkg/elastic"
)

var esConn *elasticsearch.Client

type Response struct {
	Message string `json:"message,omitempty"`
}

func main(){

	clusterURLs := flag.String("elastic-cluster", "http://localhost:9200", "Elasticsearch Cluster")
	addr := flag.String("port", ":4000", "port")
	indexName := flag.String("es-index", "products_v6", "Elasticsearch index")
	flag.Parse()

	var cfg = elasticsearch.Config{
		Addresses: []string{*clusterURLs},
	}																																																																																																																																																																																																																																																																																														
	
	conn, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}
	esConn = conn

	config := elastic.StoreConfig{Client: esConn, IndexName: *indexName}
	store, err := elastic.NewStore(config)
	if err != nil {
		log.Fatal(err)
	}
	
	r := chi.NewRouter()
	r.Get("/api/v1/suggest", func(w http.ResponseWriter, r *http.Request) {
		
		w.Header().Set("Access-Control-Allow-Origin", "*") // set up dengan periplus
		w.Header().Add("Content-Type", "application/json")

		type AutoCompleteData struct {
			ID      int    `json:"id,omitempty"`
			Name    string `json:"name,omitempty"`
			Authors string `json:"authors,omitempty"`
			ImgUrl  string `json:"img_url,omitempty"`
		}

		type AutoCompleteResponse struct {
			Response
			Data     []AutoCompleteData `json:"data,omitempty"`
		}

		var resp AutoCompleteResponse

		q := r.URL.Query().Get("q")
		f := r.URL.Query().Get("f")
		filter, err := strconv.Atoi(f)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			resp.Message = err.Error()
			m, _ := json.MarshalIndent(resp, "", "")
			w.Write(m)
			return
		}

		res, err := store.SearchAutocomplete(q, filter)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			resp.Message = err.Error()
			m, _ := json.MarshalIndent(resp, "", "")
			w.Write(m)
			return
		}

		for _, h := range res.Hits {
			id, _ := strconv.Atoi(h.ID)
			var authors string
			for _, a := range h.Contributors {
				authors += a.Name + " "
			}
			d := AutoCompleteData {
				ID: id,
				Name: authors,
				ImgUrl: h.ImgUrl,
			}
			resp.Data = append(resp.Data, d)
		}
		resp.Message = "Get data success"
		jsonResponse, err := json.MarshalIndent(resp, "", "")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			resp.Message = err.Error()
			m, _ := json.MarshalIndent(resp, "", "")
			w.Write(m)
			return
		}
		w.Write(jsonResponse)
	})

	fmt.Println("ES Cluster : " + *clusterURLs)
	fmt.Println("Listening to port : " + *addr)
	http.ListenAndServe(":" + *addr, r)
}