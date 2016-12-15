package main

import "os"
import "fmt"
import "strings"
import "net/http"
import "io/ioutil"
import "encoding/json"

func main() {

    if len(os.Args) != 2 {
        fmt.Printf("usage: %s <word>\n", os.Args[0])
        return
    }

	var url = ""
	for _, url = range []string{
		"http://ac.endic.naver.com/ac?q_enc=utf-8&st=1100&r_format=json&r_enc=utf-8",
		"http://ac.endic.naver.com/ac?q_enc=utf-8&st=11001&r_format=json&r_enc=utf-8&r_lt=10001&r_unicode=0&r_escape=1"} {

		keyword := strings.Replace(" ", "%%20", os.Args[1], -1)
		url = url + "&q=" + keyword
    
		resp, err := http.Get(url)
		if err == nil {
			body, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()

			type Completion struct {
				Query []string        `json:"query"`
				Items [][][][]string  `json:"items"`
			}

			//fmt.Println(string(body))

			var c Completion
			if err := json.Unmarshal(body, &c); err != nil {
				fmt.Println("Json decoding error!")
				return
			}
			//fmt.Printf("KEYWORD: %s\n", c.Query[0])
			//fmt.Println("COMPLETION:")
			for _, elem := range c.Items[0] {
				fmt.Printf("%s => %s\n", elem[0][0], elem[1][0])
			}

			if len(c.Items[0]) > 0 {
				return
			}
		}
	}
}
