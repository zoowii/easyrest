package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
)

// TODO: easyrest -f rpcs.json

type arrayFlags []string

func (i *arrayFlags) String() string {
	return ""
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, strings.TrimSpace(value))
	return nil
}

func sendHTTPJSONRpc(url string, method string, params *simplejson.Json, headers []string, basic *string, cookie *string) (*simplejson.Json, error) {
	if url[0:7] != "http://" {
		url = "http://" + url
	}
	bodyJSON := simplejson.New()
	bodyJSON.Set("id", 1)
	bodyJSON.Set("method", method)
	bodyJSON.Set("params", params)
	bodyStr, err := bodyJSON.Encode()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyStr)))
	if err != nil {
		return nil, err
	}
	if basic != nil {
		auth := base64.StdEncoding.EncodeToString([]byte(*basic))
		req.Header.Add("Authorization", `Basic: `+auth)
	}
	if cookie != nil {
		req.Header.Add("Cookie", *cookie)
	}
	for _, header := range headers {
		splited := strings.SplitN(header, ":", 2)
		if len(splited) >= 2 {
			req.Header.Add(splited[0], splited[1])
		}
	}
	timeout := time.Duration(5 * time.Second)
	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("http response with status code %d and body %s", resp.StatusCode, string(body))
		return nil, err
	}
	res, err := simplejson.NewJson(body)
	if err != nil {
		return nil, err
	}
	errorJSON, ok := res.CheckGet("error")
	if ok && errorJSON != nil {
		errStr, err := errorJSON.Encode()
		if err != nil {
			return nil, err
		}
		err = errors.New(string(errStr))
		return nil, err
	}
	result, ok := res.CheckGet("result")
	if !ok {
		return nil, nil
	}
	return result, nil
}

func main() {
	var basic = flag.String("basic", "", "http basic authentation user:pass")
	var cookie = flag.String("cookie", "", "cookie to send in http request")
	var headersFlags arrayFlags
	flag.Var(&headersFlags, "header", "http headers to use in request")
	flag.Parse()
	remainingArgs := flag.Args()
	if len(remainingArgs) < 3 {
		println("need pass url method params as arguments")
		os.Exit(1)
		return
	}
	url, method, paramsStr := remainingArgs[0], remainingArgs[1], remainingArgs[2]
	params, err := simplejson.NewJson([]byte(paramsStr))
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		os.Exit(1)
		return
	}
	result, err := sendHTTPJSONRpc(url, method, params, headersFlags, basic, cookie)
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		os.Exit(1)
		return
	}
	if result == nil {
		println("null")
		return
	}
	resultStr, err := result.Encode()
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		os.Exit(1)
		return
	}
	println(resultStr)
}
