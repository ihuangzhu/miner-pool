package core

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"io/ioutil"
	"math/big"
	"miner-pool/config"
	"miner-pool/jsonrpc"
	"miner-pool/util"
	"net/http"
)

type Daemon struct {
	cfg *config.Daemon

	url string
}

// Block is a block body representation
type Block struct {
	ParentHash   string         `json:"parentHash"`
	UncleHash    string         `json:"sha3Uncles"`
	Coinbase     string         `json:"miner"`
	Root         string         `json:"stateRoot"`
	TxHash       string         `json:"transactionsRoot"`
	ReceiptHash  string         `json:"receiptsRoot"`
	Bloom        string         `json:"logsBloom"`
	Difficulty   string         `json:"difficulty"`
	Number       string         `json:"number"`
	GasLimit     string         `json:"gasLimit"`
	GasUsed      string         `json:"gasUsed"`
	Time         string         `json:"timestamp"`
	Extra        string         `json:"extraData"`
	MixDigest    string         `json:"mixHash"`
	Nonce        string         `json:"nonce"`
	Hash         string         `json:"hash"`
	BaseFee      string         `json:"baseFeePerGas"`
	Transactions []*Transaction `json:"transactions"`
	Uncles       []string       `json:"uncles"`
}

// Transaction
type Transaction struct {
	Gas      string `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Hash     string `json:"hash"`
}

// TransactionReceipt
type TransactionReceipt struct {
	TxHash    string `json:"transactionHash"`
	GasUsed   string `json:"gasUsed"`
	BlockHash string `json:"blockHash"`
	Status    string `json:"status"`
}

// NewDaemon
func NewDaemon(cfg *config.Daemon) *Daemon {
	return &Daemon{
		cfg: cfg,
		url: fmt.Sprintf("http://%s:%d", *cfg.Host, *cfg.Port),
	}
}

// Coinbase delegates to `eth_coinbase` API method, and returns the miner's coinbase address
func (n *Daemon) Coinbase() (string, error) {
	data, err := n.sendHttpRequest("eth_coinbase", nil)
	if err != nil {
		return "", err
	}

	return data.(string), nil
}

// Balance delegates to `eth_getBalance` API method, and returns the address's balance
func (n *Daemon) Balance(address string) (*big.Int, error) {
	data, err := n.sendHttpRequest("eth_getBalance", []interface{}{address, "latest"})
	if err != nil {
		return nil, err
	}

	balance := big.NewInt(0).SetInt64(util.Hex2int64(data.(string)))
	return balance, nil
}

// BlockNumber delegates to `eth_blockNumber` API method, and returns the current block number
func (n *Daemon) BlockNumber() (uint64, error) {
	data, err := n.sendHttpRequest("eth_blockNumber", nil)
	if err != nil {
		return 0, err
	}

	blockNumber := util.Hex2uint64(data.(string))
	return blockNumber, nil
}

// GetBlockByNumber delegates to `eth_getBlockByNumber` RPC method, and returns block by default block parameters: "earliest", "latest" or "pending"
func (n *Daemon) GetBlockByOption(option string) (*Block, error) {
	data, err := n.sendHttpRequest("eth_getBlockByNumber", []interface{}{option, true})
	if err != nil {
		return &Block{}, err
	}

	var block Block
	err = mapstructure.Decode(data, &block)
	return &block, err
}

// GetBlockByNumber delegates to `eth_getBlockByNumber` RPC method, and returns block by number
func (n *Daemon) GetBlockByNumber(blockNumber uint64) (*Block, error) {
	data, err := n.sendHttpRequest("eth_getBlockByNumber", []interface{}{fmt.Sprintf("0x%x", blockNumber), true})
	if err != nil {
		return &Block{}, err
	}

	var block Block
	err = mapstructure.Decode(data, &block)
	return &block, err
}

// GetBlockByHash delegates to `eth_getBlockByHash` RPC method, and returns block by number
func (n *Daemon) GetBlockByHash(blockHash string) (*Block, error) {
	data, err := n.sendHttpRequest("eth_getBlockByHash", []interface{}{blockHash, true})
	if err != nil {
		return &Block{}, err
	}

	var block Block
	err = mapstructure.Decode(data, &block)
	return &block, err
}

// GetUncleByBlockNumberAndIndex delegates to `eth_getUncleByBlockNumberAndIndex` RPC method, and returns uncle by block number and index
func (n *Daemon) GetUncleByBlockNumberAndIndex(blockNumber uint64, uncleIndex int) (*Block, error) {
	data, err := n.sendHttpRequest("eth_getUncleByBlockNumberAndIndex", []interface{}{fmt.Sprintf("0x%x", blockNumber), fmt.Sprintf("0x%x", uncleIndex)})
	if err != nil {
		return &Block{}, err
	}

	var block Block
	err = mapstructure.Decode(data, &block)
	return &block, err
}

// GetUncleCountByBlockNumber delegates to `eth_getUncleCountByBlockNumber` RPC method, and returns amount of uncles by given block number
func (n *Daemon) GetUncleCountByBlockNumber(blockNumber uint64) (uint64, error) {
	data, err := n.sendHttpRequest("eth_getUncleCountByBlockNumber", []interface{}{fmt.Sprintf("0x%x", blockNumber)})
	if err != nil {
		return 0, err
	}

	uncleCount := util.Hex2uint64(data.(string))
	return uncleCount, nil
}

// GetTxReceipt
func (n *Daemon) GetTxReceipt(hash string) (*TransactionReceipt, error) {
	data, err := n.sendHttpRequest("eth_getTransactionReceipt", []string{hash})
	if err != nil {
		return &TransactionReceipt{}, err
	}

	var txReceipt TransactionReceipt
	err = mapstructure.Decode(data, &txReceipt)
	return &txReceipt, nil
}

// StratumGetWork delegates to `eth_getWork` RPC method, and returns work
func (n *Daemon) GetWork() ([]string, error) {
	data, err := n.sendHttpRequest("eth_getWork", []string{})
	if err != nil {
		return nil, err
	}

	work := data.([]interface{})
	workStrArr := make([]string, len(work))
	for i, v := range work {
		workStrArr[i] = v.(string)
	}
	return workStrArr, err
}

// StratumSubmitWork delegates to `eth_submitWork` API method, and submits work
func (n *Daemon) SubmitWork(work []string) (bool, error) {
	data, err := n.sendHttpRequest("eth_submitWork", work)
	if err != nil {
		return false, err
	}

	return data.(bool), nil
}

// sendHttpRequest 发送请求
func (n *Daemon) sendHttpRequest(method string, params interface{}) (interface{}, error) {
	req := jsonrpc.MarshalRequest(jsonrpc.Request{
		Id:      0,
		Version: jsonrpc.Version,
		Method:  method,
		Params:  params,
	})

	resp, err := http.Post(n.url, "application/json", bytes.NewBuffer(req))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Additional error check
	parsedData, err := jsonrpc.UnmarshalResponse(data)
	if err != nil {
		return nil, errors.New("Unable to unmarshal node's resp (" + string(data) + ")")
	}

	if parsedData.Error != nil {
		return nil, errors.New("Unexpected node resp: " + string(data))
	}

	return parsedData.Result, nil
}
