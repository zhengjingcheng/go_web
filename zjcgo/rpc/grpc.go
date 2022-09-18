package rpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type zjcHttpClient struct {
	client http.Client
}

func NewHttpClient() *zjcHttpClient {
	//Transport 请求分发 协程安全 连接池
	client := http.Client{
		Timeout: time.Duration(3) * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   5,
			MaxConnsPerHost:       100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return &zjcHttpClient{client: client}
}

func (c *zjcHttpClient) Get(url string, args map[string]any) ([]byte, error) {
	//get请求的参数url?
	if args != nil && len(args) > 0 {
		url = url + "?" + c.toValues(args)
	}
	log.Println(url)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.responseHandle(request)
}

func (c *zjcHttpClient) GetRequest(method string, url string, args map[string]any) (*http.Request, error) {
	if args != nil && len(args) > 0 {
		url = url + "?" + c.toValues(args)
	}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (c *zjcHttpClient) FormRequest(method string, url string, args map[string]any) (*http.Request, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(c.toValues(args)))
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (c *zjcHttpClient) PostForm(url string, args map[string]any) ([]byte, error) {
	request, err := http.NewRequest("POST", url, strings.NewReader(c.toValues(args)))
	if err != nil {
		return nil, err
	}
	return c.responseHandle(request)
}
func (c *zjcHttpClient) PostJson(url string, args map[string]any) ([]byte, error) {
	marshal, _ := json.Marshal(args)
	request, err := http.NewRequest("POST", url, bytes.NewReader(marshal))
	if err != nil {
		return nil, err
	}
	return c.responseHandle(request)
}
func (c *zjcHttpClient) JsonRequest(method string, url string, args map[string]any) (*http.Request, error) {
	jsonStr, _ := json.Marshal(args)
	req, err := http.NewRequest(method, url, bytes.NewReader(jsonStr))
	if err != nil {
		return nil, err
	}
	return req, nil
}
func (c *zjcHttpClient) Response(req *http.Request) ([]byte, error) {
	return c.responseHandle(req)
}
func (c *zjcHttpClient) responseHandle(request *http.Request) ([]byte, error) {
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		info := fmt.Sprintf("response status is %d", response.StatusCode)
		return nil, errors.New(info)
	}
	//读信息
	reader := bufio.NewReader(response.Body)
	defer response.Body.Close()
	var buf []byte = make([]byte, 127)
	var body []byte
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err == io.EOF || n == 0 {
			//读完了
			break
		}
		body = append(body, buf[:n]...)
		if n < len(buf) {
			break
		}
	}
	return body, nil

}

func (c *zjcHttpClient) toValues(args map[string]any) string {
	if args != nil && len(args) > 0 {
		params := url.Values{}
		for k, v := range args {
			params.Set(k, fmt.Sprintf("%v", v))
		}
		return params.Encode()
	}
	return ""

}

//func Get(){
//	//client request
//	client := http.Client{}
//	req := &http.Request{}
//	response, err := client.Do(req)
//	body := response.Body
//	bufio.NewReader(body)
//}
