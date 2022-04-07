package elastic

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
)

type Contributor struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Role string `json:"role,omitempty"`
}

// Document wraps an periplus book
type Document struct {
	ID           string           `json:"id,omitempty"`
	Name        string        `json:"name,omitempty"`
	ImgUrl       string        `json:"img_url,omitempty"`
	Contributors []Contributor `json:"contributors,omitempty"`
}

type SearchResults struct {
	Total int    `json:"total"`
	Hits  []*Hit `json:"hits"`
}

type Hit struct {
	Document
}

// StoreConfig configures the store
type StoreConfig struct {
	Client *elasticsearch.Client
	IndexName string
}

// Store allows to index and search documents
type Store struct {
	es *elasticsearch.Client
	indexName string
}

// NewStore returns a new instance of the store
func NewStore(c StoreConfig) (*Store, error) {
	indexName := c.IndexName
	if indexName == "" {
		indexName = "products_v6"
	}

	s := Store{es: c.Client, indexName: c.IndexName}
	return &s, nil
}

// CreateIndex creates a new index with mapping
func (s *Store) CreateIndex(mapping string) error {
	res, err := s.es.Indices.Create(s.indexName, s.es.Indices.Create.WithBody(strings.NewReader(mapping)))
	if err != nil {
		return err
	}
	if res.IsError(){
		return fmt.Errorf("error: %s", res)
	}
	return nil
}

// Exists returns true when a document with id already exists in the store.
func (s *Store) Exists(id string) (bool, error) {
	res, err := s.es.Exists(s.indexName, id)
	if err != nil {
		return false, err
	}
	switch res.StatusCode {
	case 200:
		return true, nil
	case 404:
		return false, nil
	default:
		return false, fmt.Errorf("[%s]", res.Status())
	}
}

func (s *Store) SearchAutocomplete (keyword string, filter int) (*SearchResults, error){
	var results SearchResults

	var q io.Reader
	if filter > 0 {
		q = s.BuildQuery(filter, keyword)
	}else {
		q = s.BuildQuery(-1, keyword)
	}

	res, err := s.es.Search(
		s.es.Search.WithIndex(s.indexName),
		s.es.Search.WithBody(q),
	)
	
	if err != nil {
		return &results, err
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return &results, err
		}
		return &results, fmt.Errorf("[%s] %s: %s", res.Status(), e["error"].(map[string]interface{})["type"], e["error"].(map[string]interface{})["reason"])
	}

	type envelopeResponse struct {
		Took int
		Hits struct {
			Total struct {
				Value int
			}
			Hits []struct {
				ID         string          `json:"_id"`
				Source     json.RawMessage `json:"_source"`
			}
		}
	}

	var r envelopeResponse
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return &results, nil
	}

	results.Total = r.Hits.Total.Value

	if len(r.Hits.Hits) < 1 {
		results.Hits = []*Hit{}
		return &results, nil
	}

	for _, hit := range r.Hits.Hits {
		var h Hit
		h.ID = hit.ID
		
		if err := json.Unmarshal(hit.Source, &h); err != nil {
			return &results, err
		}

		results.Hits = append(results.Hits, &h)
	}

	return &results, nil
}

func (s *Store) BuildQuery(filter int, query string) io.Reader {
	var b strings.Builder

	if filter > 0 {
		b.WriteString(fmt.Sprintf(autoCompleteWithFilter, filter, query))
	} else {
		b.WriteString(fmt.Sprintf(autoComplete, query))
	}
	// fmt.Println(b.String())
	return strings.NewReader(b.String())
}

const autoComplete =
`{
  "query": {
    "bool": {
      "must": [
        {
          "multi_match": {
            "query": %q,
            "type": "most_fields",
            "fields": [
              "type_search",
              "type_search._2gram",
              "type_search._3gram",
              "type_search._index_prefix"
            ],
            "fuzziness": 1,
            "prefix_length": 3,
            "max_expansions": 10
          }
        }
      ]
    }
  }
}
`

const autoCompleteWithFilter =
`{
  "query": {
    "bool": {
      "must": [
        {"nested": {
          "path": "categories",
          "query": {
            "bool": {
             "must": [
               {"match": {
                 "categories.id": %d
               }}
             ] 
            }
          }
        }},
        {
          "multi_match": {
            "query": %q,
            "type": "most_fields",
            "fields": [
              "type_search",
              "type_search._2gram",
              "type_search._3gram",
              "type_search._index_prefix"
            ],
            "fuzziness": 1,
            "prefix_length": 3,
            "max_expansions": 10
          }
        }
      ]
    }
  }
}
`
