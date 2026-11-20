package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func main() {

	url := "http://10.137.128.241:9004/wcs/hoister/load"
	method := "POST"
	var taskNo int
	fmt.Println("请输入任务号")
	fmt.Scanln(&taskNo)

	payload := strings.NewReader(fmt.Sprintf("{\n    \"mpCode\":\"H005\",\n    \"taskNo\":%d,\n    \"loadMode\":\"GOODS_LOAD\",\n    \"startFloor\":1,\n    \"startLocationType\":\"RIGHT\",\n    \"endFloor\":1,\n    \"endLocationType\":\"INNER\"\n}", taskNo))

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}
