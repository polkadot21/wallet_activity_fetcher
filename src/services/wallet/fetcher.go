package wallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"walletActivityParser/src/logger"
)

import (
	"walletActivityParser/src/config"
)

const (
	contentType      string = "application/json"
	rpcV             string = "2.0"
	blockNumberMth   string = "eth_blockNumber"
	blockByNumberMth string = "eth_getBlockByNumber"
	erc20Prefix      string = "0xa9059cbb"
)

type Fetcher struct {
	cfg    *config.Config
	logger *logger.Logger
}

func New(
	cfg *config.Config,
	log *logger.Logger,
) *Fetcher {
	return &Fetcher{
		cfg:    cfg,
		logger: log,
	}
}

func (f *Fetcher) blockGetCurrentBlockNumber() (int64, error) {
	payload := JsonRPCRequest{
		Jsonrpc: rpcV,
		Method:  blockNumberMth,
		Params:  []interface{}{},
		ID:      1,
	}
	payloadInfo, err := payload.String()
	if err != nil {
		return 0, err
	}
	f.logger.Printf("sending post request to %s, with payload %s", f.cfg.RpcEndpoint, payloadInfo)
	result, err := f.post(payload)
	if err != nil {
		f.logger.Printf("error sending post request to %s, error: %s", f.cfg.RpcEndpoint, err)
	}

	blockNumber, err := strconv.ParseInt(result["result"].(string), 0, 64)
	if err != nil {
		return 0, err
	}

	return blockNumber, nil
}

func (f *Fetcher) getERC20Transactions(blockNumber int64) ([]string, error) {
	// Convert the block number to hex as required by JSON-RPC
	blockNumberHex := fmt.Sprintf("0x%x", blockNumber)

	// JSON-RPC payload to get block transactions
	payload := JsonRPCRequest{
		Jsonrpc: rpcV,
		Method:  blockByNumberMth,
		Params:  []interface{}{blockNumberHex, true}, // true to get the full transaction objects
		ID:      1,
	}

	result, err := f.post(payload)
	if err != nil {
		f.logger.Printf("error sending post request to %s, error: %s", f.cfg.RpcEndpoint, err)
	}

	var addresses []string
	if result["result"] != nil {
		transactions := result["result"].(map[string]interface{})["transactions"].([]interface{})
		for _, tx := range transactions {
			transaction := tx.(map[string]interface{})
			input := transaction["input"].(string)
			// Check if the input starts with the ERC20 Transfer method signature
			if strings.HasPrefix(input, erc20Prefix) {
				// Add the "from" address
				if from, ok := transaction["from"].(string); ok && !contains(addresses, from) {
					addresses = append(addresses, from)
				}
				// Add the "to" address from the transaction, not the transfer recipient
				if to, ok := transaction["to"].(string); ok && !contains(addresses, to) {
					addresses = append(addresses, to)
				}
			}
		}
	}
	return addresses, nil
}

func (f *Fetcher) countAddressActivity(startBlock int64) map[string]int {
	activityMap := make(map[string]int)
	f.logger.Printf("fetching erc20 transactions in %v blocks & counting top %v active addresses", f.cfg.NBlocks, f.cfg.NTopWallets)
	for block := startBlock; block > startBlock-f.cfg.NBlocks; block-- {
		addresses, err := f.getERC20Transactions(block)
		if err != nil {
			f.logger.Println("Error fetching transactions for block", block, ":", err)
			continue
		}

		for _, address := range addresses {
			if _, exists := activityMap[address]; !exists {
				activityMap[address] = 1
			} else {
				activityMap[address]++
			}
		}
	}
	return activityMap
}

func (f *Fetcher) getTopNActiveAddresses(startBlock int64) ([]AddrActivity, error) {
	activityMap := f.countAddressActivity(startBlock)

	var activities []AddrActivity
	for addr, act := range activityMap {
		activities = append(activities, AddrActivity{Address: addr, Activity: act})
	}

	// Sort the slice by activity, descending
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].Activity > activities[j].Activity
	})

	// Extract the top N addresses
	f.logger.Println("sorting in descending order")
	var topNActivities []AddrActivity
	for i := 0; i < len(activities) && i < f.cfg.NTopWallets; i++ {
		topNActivities = append(topNActivities, activities[i])
	}

	return topNActivities, nil
}

func (f *Fetcher) FetchAndStore() {
	startBlock, err := f.blockGetCurrentBlockNumber()
	if err != nil {
		f.logger.Println("error fetching start block: ", err)
		panic(err)
	}
	topNActivities, err := f.getTopNActiveAddresses(startBlock)
	if err != nil {
		f.logger.Printf("error fetching top %v addresses: %s", f.cfg.NTopWallets, err)
		panic(err)
	}

	jsonData, err := json.MarshalIndent(topNActivities, "", "    ") // Using 4 spaces for indentation
	if err != nil {
		f.logger.Printf("error marshalling top addresses to JSON: %s", err)
		panic(err)
	}

	// Save JSON data to file
	if err := ioutil.WriteFile(f.cfg.SavePath, jsonData, 0644); err != nil {
		f.logger.Printf("error writing top addresses to file: %s", err)
		panic(err)
	}

	f.logger.Printf("Successfully saved top %v addresses to %s", f.cfg.NTopWallets, f.cfg.SavePath)
}

func (f *Fetcher) post(request JsonRPCRequest) (map[string]interface{}, error) {
	payloadBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Perform the HTTP POST request
	response, err := http.Post(f.cfg.RpcEndpoint, contentType, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("HTTP POST request failed: %w", err)
	}
	defer response.Body.Close()

	// Decode the JSON response
	var result map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}
