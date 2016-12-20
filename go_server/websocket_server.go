package main

import (
	"encoding/json"
	"flag"
	//"html/template"
	//"fmt"
	"log"
	"net/http"
	"net/url"
	//"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gorilla/websocket"
)

// 结构体中的变量必须大写才能被json输出 坑
type Orb struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
	//Ax       float64 `json:"ax"`
	//Ay       float64 `json:"ay"`
	Vx float64 `json:"vx"`
	Vy float64 `json:"vy"`
	Vz float64 `json:"vz"`
	//Dir      float64 `json:"dir"`
	Mass     float64 `json:"mass"`
	Size     float32 `json:"size"`
	LifeStep int     `json:"lifeStep"`
	//Color    int     `json:"color"`
	Id int `json:"id"`
	//CalcTimes int     `json:"calcTimes"`
	//flag     int     `json:"flag"`
}
type JsonRet struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data map[string]interface{} `json:"data"`
}

var addr = flag.String("addr", "0.0.0.0:8081", "websocket server address")
var mcHost = flag.String("mchost", "127.0.0.1:11211", "memcache server for reading data")

var upgrader = websocket.Upgrader{} // use default options

var mc = memcache.New(*mcHost)

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func handleOrbs(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade error:", err)
		return
	}
	defer c.Close()

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			break
		}

		//log.Printf("recv: %s", message)

		ret := JsonRet{Code: 1, Msg: "ok", Data: make(map[string]interface{})}
		//v := url.Values{}
		v, _ := url.ParseQuery(string(message))
		mcKey := v.Get("k")

		//for {

		if len(mcKey) == 0 {
			log.Println("illegal message ignored. message=", string(message))
			ret.Data["list"] = nil
		} else {
			list := getListFromMc(mc, &mcKey)
			ret.Data["list"] = list
		}
		//getListFromMc(mc, &string(message))
		//strMessage := mcKey //string(message)

		retStr, errJson := json.Marshal(ret)
		if errJson != nil {
			log.Println("json.Marshal error:", errJson)
			break
		} else {
			//log.Println("after json.Marshal retStr=", string(retStr))
		}

		err = c.WriteMessage(mt, retStr)
		if err != nil {
			log.Println("write:", err)
			break
		}

		//time.Sleep(time.Millisecond * 100)
		if len(mcKey) == 0 {
			//break
		}
		//}
		//break
	}
	log.Println("closed:", c.LocalAddr().String())
}

// 从数据库获取orbList
func getListFromMc(mc *memcache.Client, mcKey *string) (v []Orb) {
	//mapHap := make(map[string]map[string]string)
	if orbListStrVal, err := mc.Get(*mcKey); err == nil {
		//orbListStr = string(orbListStrVal.Value)
		//v = orbListStrVal.Value
		err := json.Unmarshal(orbListStrVal.Value, &v)
		if err != nil {
			log.Println("json.Unmarshal err=", err)
		}
		//fmt.Println("len(val)=", len(orbListStr), "after unmarshal, len=", len(mapHap), "err:", err)
	} else {
		log.Println("mc get(", *mcKey, ") err=", err)
	}
	return v
}

//func home(w http.ResponseWriter, r *http.Request) {
//	homeTemplate.Execute(w, "ws://"+r.Host+"/echo")
//}

func main() {
	//fmt.Println("start:")

	//return
	flag.Parse()
	log.SetFlags(0)
	//http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/echo", echo)
	http.HandleFunc("/orbs", handleOrbs)
	log.Println("server will start at", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
