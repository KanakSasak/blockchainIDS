package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"
)

func main() {

	type Payload struct {
		Jsonrpc string        `json:"jsonrpc"`
		ID      int           `json:"id"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}

	type Result struct {
		Jsonrpc string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  struct {
			Result struct {
				BlockHash            interface{}   `json:"blockHash"`
				BlockNumber          interface{}   `json:"blockNumber"`
				From                 string        `json:"from"`
				Gas                  string        `json:"gas"`
				GasPrice             string        `json:"gasPrice"`
				MaxFeePerGas         string        `json:"maxFeePerGas"`
				MaxPriorityFeePerGas string        `json:"maxPriorityFeePerGas"`
				Hash                 string        `json:"hash"`
				Input                string        `json:"input"`
				Nonce                string        `json:"nonce"`
				To                   string        `json:"to"`
				TransactionIndex     interface{}   `json:"transactionIndex"`
				Value                string        `json:"value"`
				Type                 string        `json:"type"`
				AccessList           []interface{} `json:"accessList"`
				ChainID              string        `json:"chainId"`
				V                    string        `json:"v"`
				R                    string        `json:"r"`
				S                    string        `json:"s"`
			} `json:"result"`
			Subscription string `json:"subscription"`
		} `json:"params"`
	}

	contractABI, err := abi.JSON(
		strings.NewReader(GetLocalABI("/Users/laluraynaldi/go/src/blockchainIDS/abis/VulnBank.json")),
	)
	if err != nil {
		panic(err)
	}

	messageOut := make(chan string)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	c, resp, err := websocket.DefaultDialer.Dial("wss://polygon-mumbai.g.alchemy.com/v2/vZF-bLS5x-6QJAXdpq-LNZ4S2_TZi2_P", nil)
	//c, resp, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:8545", nil)

	if err != nil {
		log.Printf("handshake failed with status %d", resp.StatusCode)
		log.Fatal("dial:", err)
	}
	//When the program closes close the connection
	defer c.Close()
	//var jsonString = `{"jsonrpc":"2.0","id": 2, "method": "eth_subscribe", "params": ["alchemy_pendingTransactions"]}`
	var jsonString = `{"jsonrpc":"2.0","id": 2, "method": "eth_subscribe", "params": ["alchemy_pendingTransactions", {"toAddress": ["0x07CDfA7A454Ef05db08c9C11261A222392EDB696"]}]}`
	//var jsonString = `{"method":"eth_getBlockByNumber","params":["pending",false],"id":1,"jsonrpc":"2.0"}`
	var jsonData = []byte(jsonString)

	var data Payload

	var errs = json.Unmarshal(jsonData, &data)
	if errs != nil {
		fmt.Println(err.Error())
		return
	}
	c.WriteJSON(data)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
			var data Result

			var errs = json.Unmarshal(message, &data)
			if errs != nil {
				fmt.Println(err.Error())
				return
			}

			//log.Printf("recv: %s", data.Params.Result.To)
			if data.Params.Result.To == "0x07CDfA7A454Ef05db08c9C11261A222392EDB696" || data.Params.Result.To == "0x07cdfa7a454ef05db08c9c11261a222392edb696" {
				log.Println("Invoked!!!")
				log.Printf("recv: %s", data.Params.Result.Input)
				DecodeTransactionInputData(&contractABI, data.Params.Result.Input)
			}

		}

	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case m := <-messageOut:
			log.Printf("Send Message %s", m)
			err := c.WriteMessage(websocket.TextMessage, []byte(m))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")
			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}

}

func GetLocalABI(path string) string {
	abiFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer abiFile.Close()

	result, err := io.ReadAll(abiFile)
	if err != nil {
		panic(err)
	}
	return string(result)
}

func DecodeTransactionInputData(contractABI *abi.ABI, data string) {
	// The first 4 bytes of the t represent the ID of the method in the ABI
	// https://docs.soliditylang.org/en/v0.5.3/abi-spec.html#function-selector
	//methodSigData := []byte("0xd0e30db0")
	datasplit := bytes.TrimLeft([]byte(data), "0x")
	datax, err := hex.DecodeString(string(datasplit[:]))
	//methodSigData := datax[4:]
	method, err := contractABI.MethodById(datax)
	if err != nil {
		panic(err)
	}

	//inputsSigData := datax[4:]
	//inputsMap := make(map[string]interface{})
	//if err := method.Inputs.UnpackIntoMap(inputsMap, inputsSigData); err != nil {
	//	panic(err)
	//}

	fmt.Printf("Method Name: %s\n", method.Name)
	//fmt.Printf("Method inputs: %v\n", inputsMap)

	//encodedData := "d0e30db0"
	//decodeData, err := hex.DecodeString(encodedData)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//// a9059cbb == transfer
	//if method, ok := contractABI.Methods["deposit"]; ok {
	//	params, err := method.Inputs.Unpack(decodeData[4:])
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	log.Println(params)
	//	log.Println(ok)
	//
	//}

}
