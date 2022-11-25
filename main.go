package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Server struct {
	host string
	port int
}

type Client struct {
	conn net.Conn
}

type NodeRequest struct {
	JsonRpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Id      int    `json:"id"`
}

type NodeResponse struct {
	JsonRpc string `json:"jsonrpc"`
	Result  string `json:"result"`
	Id      int    `json:"id"`
}

func main() {
	host := getEnv("HOST", "localhost")
	port, _ := strconv.Atoi(getEnv("PORT", "1337"))
	server := Server{host: host, port: port}
	server.Run()
}

func (server *Server) Run() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%v", server.host, server.port))
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	fmt.Printf("Listening on %v:%v\n", server.host, server.port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		client := &Client{
			conn: conn,
		}
		go client.handleRequest()
	}
}

func (client *Client) handleRequest() {
	_, err := bufio.NewReader(client.conn).ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return
	}
	client.conn.Write([]byte("\n"))
	isHealthy := checkHealth()
	if isHealthy {
		client.conn.Write([]byte("up\n"))
	} else {
		client.conn.Write([]byte("down\n"))
	}
	client.conn.Close()
}

func checkHealth() bool {
	maxHeightDiff, _ := strconv.ParseInt(getEnv("MAX_HEIGHT_DIFF", "100"), 10, 64)
	referenceNodeUrl := getEnv("REFERENCE_NODE_URL", "localhost:8545")
	nodeUrl := getEnv("NODE_URL", "localhost:8545")

	referenceBlockNumber := getLatestBlock(referenceNodeUrl)
	blockNumber := getLatestBlock(nodeUrl)
	diff := int64(math.Abs(float64(referenceBlockNumber - blockNumber)))
	return diff <= maxHeightDiff
}

func getLatestBlock(nodeUrl string) int64 {
	payload := NodeRequest{
		JsonRpc: "2.0",
		Method:  "eth_blockNumber",
		Id:      1,
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(nodeUrl, "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("Error! Could not request latest block from node.")
		return 0
	}
	defer resp.Body.Close()

	if resp.Status != "200 OK" {
		fmt.Printf("Error! Received HTTP status %v from node %v\n", resp.Status, nodeUrl)
		return 0
	}
	responseBody, _ := ioutil.ReadAll(resp.Body)
	var nodeResponse NodeResponse
	if err := json.Unmarshal(responseBody, &nodeResponse); err != nil {
		fmt.Println("Error! Cannot unmarshal JSON")
		return 0
	}

	var blockNumber int64
	if strings.HasPrefix(nodeResponse.Result, "0x") {
		blockNumber, _ = strconv.ParseInt(nodeResponse.Result, 0, 64)
	} else {
		blockNumber, _ = strconv.ParseInt(nodeResponse.Result, 10, 64)
	}
	return blockNumber
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
