package wallet

import (
	"encoding/json"
	"fmt"
)

type JsonRPCRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

func (jr JsonRPCRequest) String() (string, error) {
	paramsJSON, err := json.Marshal(jr.Params)
	if err != nil {
		return "", fmt.Errorf("failed marshaling params: %w", err)
	}
	return fmt.Sprintf("Jsonrpc: %s, method: %s, params: %s, id: %d", jr.Jsonrpc, jr.Method, string(paramsJSON), jr.ID), nil
}

type AddrActivity struct {
	Address  string `json:"address"`
	Activity int    `json:"activity"`
}
