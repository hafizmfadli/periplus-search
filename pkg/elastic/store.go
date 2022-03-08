package elastic

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
)

// Document wraps an periplus product
type Document struct {
	ID             int    `json:"id,omitempty"`
	FirstAuthor    string `json:"first_author,omitempty"`
	SecondAuthor   string `json:"second_author,omitempty"`
	ThirdAuthor    string `json:"third_author,omitempty"`
	ISBN           string `json:"isbn,omitempty"`
	Title          string `json:"title,omitempty"`
	ImgUrl         string `json:"img_url,omitempty"`
	AuthorsSuggest string `json:"authors_suggest,omitempty"`
	TitleSuggest   string `json:"title_suggest,omitempty"`
}

type SearchResults struct {
	// Total int    `json:"total"`
	// Hits  []*Hit `json:"hits"`
	Took int64 `json:"took,omitempty"`
	Hits struct {
		Total struct {
			Value int64 `json:"value,omitempty"`
		} `json:"total,omitempty"`
		Hits []Hit `json:"hits,omitempty"`
	} `json:"hits,omitempty"`
}

type Hit struct {
	Doc        Document `json:"_source,omitempty"`
	Highlights struct {
		AuthorsSuggest []string `json:"authors_suggest,omitempty"`
		TitleSuggest   []string `json:"title_suggest,omitempty"`
	} `json:"highlight,omitempty"`
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
		indexName = "products"
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

func (s *Store) SearchAutocomplete (keyword string) (*SearchResults, error){
	var results SearchResults

	res, err := s.es.Search(
		s.es.Search.WithIndex(s.indexName),
		s.es.Search.WithBody(s.buildQuery(autoComplete, keyword)),
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

	if err := json.NewDecoder(res.Body).Decode(&results); err != nil {
		return &results, err
	}
	
	return &results, nil
}

func (s *Store) buildQuery(queryTemplate, queryValue string) io.Reader {
	var b strings.Builder

	if queryTemplate == "" {
		b.WriteString(searchAll)
	}else {
		b.WriteString(fmt.Sprintf(queryTemplate, queryValue))
	}

	return strings.NewReader(b.String())
}

const searchAll = `
	"query" : { "match_all" : {} },
	"size" : 10
`

const autoComplete = `
{
  "query": {
    "multi_match": {
      "query": %q,
      "type": "cross_fields", 
      "fields": ["title_suggest", "authors_suggest"],
      "operator": "and"
    }
  },
  "highlight": {
    "fields": {
      "title_suggest": {},
      "authors_suggest": {}
    }
  }
}
`



