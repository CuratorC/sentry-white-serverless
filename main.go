package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aliyun/fc-runtime-go-sdk/fc"
	"io"
	"io/ioutil"
	"net/http"
)

func main() {
	fc.StartHttp(HandleHttpRequest)
}

// HandleHttpRequest ...
func HandleHttpRequest(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	fmt.Println(req.Body)
	s := "body: "
	_, err := io.WriteString(w, s)
	if err != nil {
		return err
	}

	return nil
}

//postForm 获取 post form 形式的参数
func postForm(req *http.Request) map[string]string {
	//body, _ := ioutil.ReadAll(req.Body)
	var result = make(map[string]string)
	req.ParseForm()
	for k, v := range req.PostForm {
		if len(v) < 1 {
			continue
		}

		result[k] = v[0]
	}

	return result
}

//postJson 获取 post json 参数
func postJson(req *http.Request, obj interface{}) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, obj)
	if err != nil {
		return err
	}

	return nil
}
