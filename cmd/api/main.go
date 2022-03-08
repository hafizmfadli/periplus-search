package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/go-chi/chi/v5"
	"github.com/hafizmfadli/periplus-search/pkg/elastic"
)

var esConn *elasticsearch.Client

// var cfg = elasticsearch.Config{
// 	CloudID: os.Getenv("ELASTIC_CLOUD_ID"),
// 	Username: os.Getenv("ELASTIC_CLOUD_USERNAME"),
// 	Password: os.Getenv("ELASTIC_CLOUD_PASSWORD"),
// }

var cfg = elasticsearch.Config{
	CloudID: "messing-around:YXAtc291dGhlYXN0LTEuYXdzLmZvdW5kLmlvJGUxNjJjNTU3NWY5ZTQxZTVhZDVkODczNmYyOGVhOTM4JGVhOTA5NjcwNmNmNDQzNGE4NTUwNTgzNGE4OGYxYjRh",
	Username: "elastic",
	Password: "zVzVwY7tcTUxGjfPZfuNe5fj",
}


type SearchResponse struct {
	Took int64
	Hits struct {
		Total struct {
			Value int64
		}
		Hits []interface{}
	}
}

func main(){
	port := os.Getenv("PORT")
	conn, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}
	esConn = conn

	config := elastic.StoreConfig{Client: esConn, IndexName: "products-v6"}
	store, err := elastic.NewStore(config)
	if err != nil {
		log.Fatal(err)
	}
	
	r := chi.NewRouter()
	r.Get("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		queryParams := r.URL.Query().Get("q")

		res, err := store.SearchAutocomplete(queryParams)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		
		type SuggestionResponse struct {
			ImageUrl       string `json:"image_url,omitempty"`
			Title          string `json:"title,omitempty"`
			Authors        string `json:"authors,omitempty"`
			TitleSuggest   string `json:"title_suggest,omitempty"`
			AuthorsSuggest string `json:"authors_suggest,omitempty"`
		}

		 suggestion := map[string][]SuggestionResponse{}
		for _, h := range res.Hits.Hits {

			a := h.Doc.AuthorsSuggest
			if len(h.Highlights.AuthorsSuggest) > 0 {
				a = h.Highlights.AuthorsSuggest[0]
			}

			t := h.Doc.TitleSuggest
			if len(h.Highlights.TitleSuggest) > 0 {
				t = h.Highlights.TitleSuggest[0]
			}

			temp := SuggestionResponse {
				ImageUrl: h.Doc.ImgUrl,
				Title: h.Doc.Title,
				Authors: h.Doc.AuthorsSuggest,
				TitleSuggest: t,
				AuthorsSuggest: a,
			}
			suggestion["data"] = append(suggestion["data"], temp)
		}
		jsonResponse, err := json.Marshal(suggestion)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write(jsonResponse)
	})

	fmt.Println("Listening to port : " + port)
	http.ListenAndServe(":" + port, r)
}