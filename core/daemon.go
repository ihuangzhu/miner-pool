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
	"time"
)

type Daemon struct {
	cfg *config.Daemon

	url string
}

// Block is a block body representation
type Block struct {
	ParentHash   string         `mapstructure:"parentHash"`
	UncleHash    string         `mapstructure:"sha3Uncles"`
	Coinbase     string         `mapstructure:"miner"`
	Root         string         `mapstructure:"stateRoot"`
	TxHash       string         `mapstructure:"transactionsRoot"`
	ReceiptHash  string         `mapstructure:"receiptsRoot"`
	Bloom        string         `mapstructure:"logsBloom"`
	Difficulty   string         `mapstructure:"difficulty"`
	Number       string         `mapstructure:"number"`
	GasLimit     string         `mapstructure:"gasLimit"`
	GasUsed      string         `mapstructure:"gasUsed"`
	Time         string         `mapstructure:"timestamp"`
	Extra        string         `mapstructure:"extraData"`
	MixDigest    string         `mapstructure:"mixHash"`
	Nonce        string         `mapstructure:"nonce"`
	Hash         string         `mapstructure:"hash"`
	BaseFee      string         `mapstructure:"baseFeePerGas"`
	Transactions []*Transaction `mapstructure:"transactions"`
	Uncles       []string       `mapstructure:"uncles"`
}

// Transaction
type Transaction struct {
	Gas      string `mapstructure:"gas"`
	GasPrice string `mapstructure:"gasPrice"`
	Hash     string `mapstructure:"hash"`
}

// TransactionReceipt
type TransactionReceipt struct {
	TxHash    string `mapstructure:"transactionHash"`
	GasUsed   string `mapstructure:"gasUsed"`
	BlockHash string `mapstructure:"blockHash"`
	Status    string `mapstructure:"status"`
}

// NewDaemon
func NewDaemon(cfg *config.Daemon) *Daemon {
	return &Daemon{
		cfg: cfg,
		url: fmt.Sprintf("http://%s:%d", *cfg.Host, *cfg.Port),
	}
}

// PeerCount
func (d *Daemon) PeerCount() (uint64, error) {
	data, err := d.sendHttpRequest("net_peerCount", nil)
	if err != nil {
		return 0, err
	}

	peerCount := util.Hex2uint64(data.(string))
	return peerCount, nil
}

// Coinbase delegates to `eth_coinbase` API method, and returns the miner's coinbase address
func (d *Daemon) Coinbase() (string, error) {
	data, err := d.sendHttpRequest("eth_coinbase", nil)
	if err != nil {
		return "", err
	}

	return data.(string), nil
}

// Balance delegates to `eth_getBalance` API method, and returns the address's balance
func (d *Daemon) Balance(address string) (*big.Int, error) {
	data, err := d.sendHttpRequest("eth_getBalance", []interface{}{address, "latest"})
	if err != nil {
		return nil, err
	}

	balance := big.NewInt(0).SetInt64(util.Hex2int64(data.(string)))
	return balance, nil
}

// BlockNumber delegates to `eth_blockNumber` API method, and returns the current block number
func (d *Daemon) BlockNumber() (uint64, error) {
	data, err := d.sendHttpRequest("eth_blockNumber", nil)
	if err != nil {
		return 0, err
	}

	blockNumber := util.Hex2uint64(data.(string))
	return blockNumber, nil
}

// GetBlockByNumber delegates to `eth_getBlockByNumber` RPC method, and returns block by default block parameters: "earliest", "latest" or "pending"
func (d *Daemon) GetBlockByOption(option string) (*Block, error) {
	data, err := d.sendHttpRequest("eth_getBlockByNumber", []interface{}{option, true})
	if err != nil {
		return &Block{}, err
	}

	var block Block
	err = mapstructure.Decode(data, &block)
	return &block, err
}

// GetBlockByNumber delegates to `eth_getBlockByNumber` RPC method, and returns block by number
func (d *Daemon) GetBlockByNumber(blockNumber uint64) (*Block, error) {
	data, err := d.sendHttpRequest("eth_getBlockByNumber", []interface{}{fmt.Sprintf("0x%x", blockNumber), true})
	if err != nil {
		return &Block{}, err
	}

	var block Block
	err = mapstructure.Decode(data, &block)
	return &block, err
}

// GetBlockByHash delegates to `eth_getBlockByHash` RPC method, and returns block by number
func (d *Daemon) GetBlockByHash(blockHash string) (*Block, error) {
	data, err := d.sendHttpRequest("eth_getBlockByHash", []interface{}{blockHash, true})
	if err != nil {
		return &Block{}, err
	}

	var block Block
	err = mapstructure.Decode(data, &block)
	return &block, err
}

// GetUncleByBlockNumberAndIndex delegates to `eth_getUncleByBlockNumberAndIndex` RPC method, and returns uncle by block number and index
func (d *Daemon) GetUncleByBlockNumberAndIndex(blockNumber uint64, uncleIndex int) (*Block, error) {
	data, err := d.sendHttpRequest("eth_getUncleByBlockNumberAndIndex", []interface{}{fmt.Sprintf("0x%x", blockNumber), fmt.Sprintf("0x%x", uncleIndex)})
	if err != nil {
		return &Block{}, err
	}

	var block Block
	err = mapstructure.Decode(data, &block)
	return &block, err
}

// GetUncleCountByBlockNumber delegates to `eth_getUncleCountByBlockNumber` RPC method, and returns amount of uncles by given block number
func (d *Daemon) GetUncleCountByBlockNumber(blockNumber uint64) (uint64, error) {
	data, err := d.sendHttpRequest("eth_getUncleCountByBlockNumber", []interface{}{fmt.Sprintf("0x%x", blockNumber)})
	if err != nil {
		return 0, err
	}

	uncleCount := util.Hex2uint64(data.(string))
	return uncleCount, nil
}

// GetTxReceipt
func (d *Daemon) GetTxReceipt(hash string) (*TransactionReceipt, error) {
	data, err := d.sendHttpRequest("eth_getTransactionReceipt", []string{hash})
	if err != nil {
		return &TransactionReceipt{}, err
	}

	var txReceipt TransactionReceipt
	err = mapstructure.Decode(data, &txReceipt)
	return &txReceipt, nil
}

<<<<<<< Updated upstream
// GetNetworkHashrate
func (d *Daemon) GetNetworkHashrate(sampleSize uint64) (uint64, error) {
	// 往前数块数
	if sampleSize == 0 {
		sampleSize = 3800
	}

	// 当前块信息
	latestBlock, err := d.GetBlockByOption("latest")
=======
// GetNetworkHashrate Calculating hashrate of window in seconds
func (d *Daemon) GetNetworkHashrate(window uint64) (uint64, error) {
	// 最新块信息
	endTimestamp := uint64(time.Now().Unix())
	lastBlockNumber, err := d.BlockNumber()
>>>>>>> Stashed changes
	if err != nil {
		return 0, err
	}

<<<<<<< Updated upstream
	// 往前数块信息
	latestBlockNumber := util.Hex2uint64(latestBlock.Number) // Save this value to atomically get a block number.
	targetBlockNum := latestBlockNumber - sampleSize
	targetBlock, err := d.GetBlockByNumber(targetBlockNum)
	if err != nil {
		return 0, err
	}

	// 计算网络难度
	startTimestamp := util.Hex2uint64(latestBlock.Time)
	endTimestamp := util.Hex2uint64(targetBlock.Time)
	blockTime := (startTimestamp - endTimestamp) / sampleSize
	difficulty := util.Hex2uint64(latestBlock.Difficulty) // You can sum up the last n-blocks and average; this is mathematically sound.

	return difficulty / blockTime, nil
=======
	startTimestamp := endTimestamp
	blockNumber := lastBlockNumber

	// 往前推到指定时间
	var difficulty uint64
	for startTimestamp > endTimestamp-window {
		targetBlock, _ := d.GetBlockByNumber(blockNumber)
		startTimestamp = util.Hex2uint64(targetBlock.Time)
		difficulty += util.Hex2uint64(targetBlock.Difficulty)

		blockNumber--
	}

	// 计算难度
	hashrate := difficulty / (endTimestamp - startTimestamp)
	return hashrate, nil
>>>>>>> Stashed changes
}

// StratumGetWork delegates to `eth_getWork` RPC method, and returns work
func (d *Daemon) GetWork() ([]string, error) {
	data, err := d.sendHttpRequest("eth_getWork", []string{})
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
func (d *Daemon) SubmitWork(work []string) (bool, error) {
	data, err := d.sendHttpRequest("eth_submitWork", work)
	if err != nil {
		return false, err
	}

	return data.(bool), nil
}

// sendHttpRequest 发送请求
func (d *Daemon) sendHttpRequest(method string, params interface{}) (interface{}, error) {
	req := jsonrpc.MarshalRequest(jsonrpc.Request{
		Id:      0,
		Version: jsonrpc.Version,
		Method:  method,
		Params:  params,
	})

	resp, err := http.Post(d.url, "application/json", bytes.NewBuffer(req))
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
