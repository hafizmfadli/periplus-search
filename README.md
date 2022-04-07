# Periplus's Search

Prerequisite : [Go 1.17](https://go.dev/dl/)

## How to run this project

### 1.Install all dependecies
```
go get -u ./...
```

### 2.Run project
```
go run cmd/api/*.go -elastic-cluster $YOUR_ELASTIC_CLUSTER_URL -es-index $YOUR_ELASTIC_INDEX_NAME -port $YOUR_PORT
```

example :
```
go run cmd/api/*.go -elastic-cluster http://localhost:9500 -es-index products_v6 -port 5000
```

### Notes

Parameter default value :

elasticsearch cluster url : http://localhost:9200 (default)

elasticsearch index name : products_v6 (default)

port number : 4000 (default)

### How to get help :
```
go run cmd/api/*.go -help
```


