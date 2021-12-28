package jsonrpc

import "encoding/json"

const Version = "2.0"

type Request struct {
	Id      int         `json:"id"`
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type RequestStratum struct {
	Request

	Params []string `json:"params"`
	Worker string   `json:"worker"`
}

type Response struct {
	Id      int         `json:"id"`
	Version string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error"`
}

func UnmarshalRequest(b []byte) (RequestStratum, error) {
	var req RequestStratum
	err := json.Unmarshal(b, &req)
	return req, err
}

func UnmarshalResponse(b []byte) (Response, error) {
	var resp Response
	err := json.Unmarshal(b, &resp)
	return resp, err
}

func MarshalResponse(r Response) []byte {
	resp, _ := json.Marshal(r)
	return resp
}

func MarshalRequest(r Request) []byte {
	req, _ := json.Marshal(r)
	return req
}
