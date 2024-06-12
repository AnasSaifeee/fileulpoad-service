package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dustin/go-humanize"
	opensearch "github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchutil"
)
var (
	service *OpenSearchService
)

type OpenSearchService struct {
	client *opensearch.Client
}




func NewOpenSearchService() (*OpenSearchService, error) {
	if service == nil {
		service = &OpenSearchService{}
		client, err := CreateOpenSearchClient()
		if err != nil {
			return nil, err
		}
		service.client = client
	}
	return service, nil
}

// CreateElasticClient - Client Creation for elastic
func CreateOpenSearchClient() (*opensearch.Client, error) {
	openSearchHost := fmt.Sprintf(
		"%v:%v",
		// utils.GetConfig().Elastic.Host,
		// utils.GetConfig().Elastic.Port,
		"https://opensearch.vectoredge.io",
		"9200",
	)


	client, err := opensearch.NewClient(opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Addresses: []string{openSearchHost},
		// Username:  utils.GetConfig().Elastic.Username,
		// Password:  utils.GetConfig().Elastic.Password,
		Username: "admin",
		Password:"Rahul@1300",
	})

	if err != nil {
		fmt.Sprintf("\n\ncannot initialize opensearch go client: %v\n\n", err)
		return nil, err
	}

	res, err := client.Info()
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}

	log.Println(res)

	return client, err
}


func (es *OpenSearchService) CreateBulkClassificationRecords(
	classificationRecords []ClassificationLogModel,
) error {
	indexName := "ve-classification-1"
	indexer, err := opensearchutil.NewBulkIndexer(opensearchutil.BulkIndexerConfig{
		Client:     es.client, // The Elasticsearch client
		Index:      indexName, // The default index name
		NumWorkers: 1,         // The number of worker goroutines (default: number of CPUs)
		FlushBytes: 1e+2,      // The flush threshold in bytes (default: 5M)
	})
	if err != nil {
		log.Printf("Issue in creating Indexer")
		return err
	}
	startTime := time.Now().UTC()
	for _, record := range classificationRecords {
		// Prepare the data payload: encode article to JSON
		//
		data, err := json.Marshal(record)
		if err != nil {
			log.Printf("Cannot encode Access record %v: %s", record, err)
		}

		err = indexer.Add(
			context.Background(),

			opensearchutil.BulkIndexerItem{
				// Action field configures the operation to perform (index, create, delete, update)
				Action: "index",

				// Body is an `io.Reader` with the payload
				Body: bytes.NewReader(data),

				// OnSuccess is called for each successful operation
				OnSuccess: func(ctx context.Context, item opensearchutil.BulkIndexerItem, res opensearchutil.BulkIndexerResponseItem) {
					fmt.Sprintf("\n\nInside on Success for item: %+v", item)
				},
				// OnSuccess: nil,
				// OnFailure: nil,
				// OnFailure is called for each failed operation
				OnFailure: func(ctx context.Context, item opensearchutil.BulkIndexerItem, res opensearchutil.BulkIndexerResponseItem, err error) {
					if err != nil {
						log.Printf("ERROR: %s", err)
					} else {
						log.Printf("ERROR: %s: %s", res.Error.Type, res.Error.Reason)
					}
				},
			},
		)
		if err != nil {
			log.Fatalf("Unexpected error: %s", err)
		}

	}

	if err := indexer.Close(context.Background()); err != nil {
		log.Fatalf("Unexpected error: %s", err)
	}

	biStats := indexer.Stats()

	// Report the results: number of indexed docs, number of errors, duration, indexing rate
	//
	dur := time.Since(startTime)

	if biStats.NumFailed > 0 {
		log.Printf(
			"\nIndexed [%s] documents for index %v with [%s] errors in %s (%s docs/sec)",
			humanize.Comma(int64(biStats.NumFlushed)),
			indexName,
			humanize.Comma(int64(biStats.NumFailed)),
			dur.Truncate(time.Millisecond),
			humanize.Comma(int64(1000.0/float64(dur/time.Millisecond)*float64(biStats.NumFlushed))),
		)
	} else {
		log.Printf(
			"\nSucessfuly indexed [%s] documents for index %v in %s (%s docs/sec)",
			humanize.Comma(int64(biStats.NumFlushed)),
			indexName,
			dur.Truncate(time.Millisecond),
			humanize.Comma(int64(1000.0/float64(dur/time.Millisecond)*float64(biStats.NumFlushed))),
		)
	}

	return nil
}