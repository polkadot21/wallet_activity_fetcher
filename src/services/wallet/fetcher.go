package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/time/rate"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
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
	cfg     *config.Config
	logger  *logger.Logger
	limiter *rate.Limiter
}

func New(
	cfg *config.Config,
	log *logger.Logger,
) *Fetcher {
	fetcher := Fetcher{
		cfg:    cfg,
		logger: log,
	}
	fetcher.InitRateLimiter(10, 10)
	return &fetcher
}

func (f *Fetcher) blockGetCurrentBlockNumber() (int64, error) {
	payload := JsonRPCRequest{
		Jsonrpc: rpcV,
		Method:  blockNumberMth,
		Params:  []interface{}{},
		ID:      1,
	}

	result, err := f.post(payload)
	if err != nil {
		f.logger.Errorf("error sending post request to %s, error: %s", f.cfg.RpcEndpoint, err)
	}

	blockNumber, err := strconv.ParseInt(result["result"].(string), 0, 64)
	if err != nil {
		f.logger.Errorf("error parsing str block number %s as int, error: %s", result["result"], err)
		return 0, err
	}
	f.logger.Infof("parsed block number %v", blockNumber)
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
		f.logger.Errorf("error sending post request to %s, error: %s", f.cfg.RpcEndpoint, err)
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
				if from, ok := transaction["from"].(string); ok {
					addresses = append(addresses, from)
				}
				// Add the "to" address from the transaction, not the transfer recipient
				if to, ok := transaction["to"].(string); ok {
					addresses = append(addresses, to)
				}
			}
		}
	}
	return addresses, nil
}

func (f *Fetcher) countAddressActivity(startBlock int64) map[string]int {
	activityMap := make(map[string]int)
	var mu sync.Mutex // Used to safely update activityMap from multiple goroutines

	f.logger.Infof("fetching erc20 transactions in %v blocks & counting top %v active addresses", f.cfg.NBlocks, f.cfg.NTopWallets)

	startTime := time.Now()
	var wg sync.WaitGroup // WaitGroup to wait for all goroutines to finish

	// Channel to receive addresses from goroutines
	addressChan := make(chan []string, f.cfg.NBlocks)

	for block := startBlock; block > startBlock-f.cfg.NBlocks; block-- {
		wg.Add(1)
		go func(b int64) {
			defer wg.Done()
			addresses, err := f.getERC20Transactions(b)
			if err != nil {
				f.logger.Errorf("Error fetching transactions for block %d: %v", b, err)
				return
			}
			addressChan <- addresses
		}(block)
	}

	// Close the channel once all goroutines have finished
	go func() {
		wg.Wait()
		close(addressChan)
	}()

	// Collect addresses from the channel and update the activityMap
	for addresses := range addressChan {
		for _, address := range addresses {
			mu.Lock()
			if _, exists := activityMap[address]; !exists {
				activityMap[address] = 1
			} else {
				activityMap[address]++
			}
			mu.Unlock()
		}
	}

	duration := time.Since(startTime)
	f.logger.Infof("Completed fetching and processing in %s", duration)

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
	f.logger.Infof("sorting in descending order")
	var topNActivities []AddrActivity
	for i := 0; i < len(activities) && i < f.cfg.NTopWallets; i++ {
		topNActivities = append(topNActivities, activities[i])
	}

	return topNActivities, nil
}

func (f *Fetcher) FetchAndStore() {
	startBlock, err := f.blockGetCurrentBlockNumber()
	if err != nil {
		f.logger.Errorf("error fetching start block: ", err)
		panic(err)
	}
	topNActivities, err := f.getTopNActiveAddresses(startBlock)
	if err != nil {
		f.logger.Errorf("error fetching top %v addresses: %s", f.cfg.NTopWallets, err)
		panic(err)
	}

	jsonData, err := json.MarshalIndent(topNActivities, "", "    ") // Using 4 spaces for indentation
	if err != nil {
		f.logger.Errorf("error marshalling top addresses to JSON: %s", err)
		panic(err)
	}

	// Save JSON data to file
	if err = ioutil.WriteFile(f.cfg.SavePath, jsonData, 0644); err != nil {
		f.logger.Errorf("error writing top addresses to file: %s", err)
		panic(err)
	}

	f.logger.Infof("Successfully saved top %v addresses to %s", f.cfg.NTopWallets, f.cfg.SavePath)
}

func (f *Fetcher) InitRateLimiter(rps float64, burst int) {
	f.limiter = rate.NewLimiter(rate.Limit(rps), burst)
}

func (f *Fetcher) post(request JsonRPCRequest) (map[string]interface{}, error) {
	err := f.limiter.Wait(context.Background())
	if err != nil {
		f.logger.Errorf("Rate limiter: %v", err)
		return nil, err
	}

	payloadInfo, err := request.String()
	if err != nil {
		f.logger.Errorf("failed creating payload info: %s", err)
		return nil, err
	}
	f.logger.Infof("sending post request to %s, with payload %s", f.cfg.RpcEndpoint, payloadInfo)

	payloadBytes, err := json.Marshal(request)
	if err != nil {
		f.logger.Errorf("failed to marshal request: %w", err)
		return nil, err
	}

	// Perform the HTTP POST request
	response, err := http.Post(f.cfg.RpcEndpoint, contentType, bytes.NewBuffer(payloadBytes))
	if err != nil {
		f.logger.Errorf("HTTP POST request failed: %w", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		// Read the response body
		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			f.logger.Errorf("HTTP POST request returned non-OK status: %d, but response body could not be read: %v", response.StatusCode, err)
			return nil, fmt.Errorf("HTTP POST request returned non-OK status: %d, but response body could not be read: %v", response.StatusCode, err)
		}

		responseBody := string(bodyBytes)

		f.logger.Errorf("HTTP POST request returned non-OK status: %d, response: %s", response.StatusCode, responseBody)

		return nil, fmt.Errorf("HTTP POST request returned non-OK status: %d, response: %s", response.StatusCode, responseBody)
	}

	// Decode the JSON response
	var result map[string]interface{}
	if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
		f.logger.Errorf("failed to decode response: %w", err)
		return nil, err
	}

	return result, nil
}
