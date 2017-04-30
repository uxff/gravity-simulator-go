package main

import (
	"encoding/json"
	"flag"
	//"fmt"
	"log"
	"net/http"
	"net/url"
	//"time"
	orbs "./orbs"
	saverpkg "./saver"

	//"github.com/bradfitz/gomemcache/memcache"
	"github.com/gorilla/websocket"
)

// json返回值结构
type JsonRet struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data map[string]interface{} `json:"data"`
}

// 使用精简数据格式传输，提高网络使用率，降低chrome内存使用，使chrome支持100W粒子
type TinyOrb struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
	M float32 `json:"m"`
	//Stat int     `json:"st"`
}

var addr = flag.String("addr", "0.0.0.0:8081", "websocket server address")

var upgrader = websocket.Upgrader{} // use default options

var saver = saverpkg.Saver{}

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

func ToTinyOrbList(list []orbs.Orb) []TinyOrb {
	olist := make([]TinyOrb, len(list))
	for i := 0; i < len(list); i++ {
		o := &olist[i]
		t := &list[i]
		//o.Stat = int(t.Stat)
		if t.Stat == 1 {
			o.X = float32(t.X)
			o.Y = float32(t.Y)
			o.Z = float32(t.Z)
			o.M = float32(t.Mass)
		}
	}
	return olist
}

func handleOrbs(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade error:", err)
		return
	}
	defer c.Close()

	log.Println("got a client:", r.RemoteAddr, r.URL)

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

		if len(mcKey) == 0 {
			log.Println("illegal message ignored. message=", string(message))
			ret.Data["list"] = nil
		} else {
			list := saver.GetList(&mcKey) //getListFromMc(mc, &mcKey)
			tinyList := ToTinyOrbList(list)
			ret.Data["list"] = tinyList
		}

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
		//break
	}
	log.Println("closed:", c.LocalAddr().String())
}

//func home(w http.ResponseWriter, r *http.Request) {
//	homeTemplate.Execute(w, "ws://"+r.Host+"/echo")
//}

func main() {
	var savePath = flag.String("savepath", "mc://127.0.0.1:11211", "where to save, support mc/file/redis\n\tlike: file://./filecache/")
	flag.Parse()

	saver.SetSavepath(savePath)

	log.SetFlags(log.Lshortfile | log.LstdFlags)

	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/echo", echo)
	http.HandleFunc("/orbs", handleOrbs)
	log.Println("server will start at", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
