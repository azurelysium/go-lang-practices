package main

import "os"
import "fmt"
import "net/http"
import "io/ioutil"
import "encoding/json"

func main() {

    if len(os.Args) != 2 {
        fmt.Printf("usage: %s <word>\n", os.Args[0])
        return
    }

    var url = "http://ac.endic.naver.com/ac?q_enc=utf-8&st=1100&r_format=json&r_enc=utf-8"
    url = url + "&q=" + os.Args[1]
    
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
    }
}
