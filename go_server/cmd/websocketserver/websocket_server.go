package main

import (
	"encoding/json"
	"flag"

	//"fmt"
	"log"
	"net/http"
	"net/url"

	//"time"
	"github.com/uxff/gravity-simulator-go/go_server/orbs"
	"github.com/uxff/gravity-simulator-go/go_server/saver"

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

var theSaver = saver.Saver{}

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

/*将json数据包缩小50%*/
func ToTinyOrbList(list []orbs.Orb) []TinyOrb {
	olist := make([]TinyOrb, len(list))
	for i := 0; i < len(list); i++ {
		o := &olist[i]
		t := &list[i]
		//o.Stat = int(t.Stat)
		if t.Id > 0 {
			o.X = float32(t.X)
			o.Y = float32(t.Y)
			o.Z = float32(t.Z)
			o.M = float32(t.Mass)
		}
	}
	return olist
}

/*展示用数据，包括:x,y,z,m,对应下标0,1,2,3. 此种格式缩小到ToTinyOrbList格式的40%*/
func ToFloatList(list []orbs.Orb) [][4]float32 {
	olist := make([][4]float32, len(list))
	for i := 0; i < len(list); i++ {
		o := &olist[i]
		t := &list[i]
		if t.Id > 0 {
			o[0] = float32(t.X)
			o[1] = float32(t.Y)
			o[2] = float32(t.Z)
			o[3] = float32(t.Mass)
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

	q := r.URL.Query()
	savePath := q.Get("savepath")

	if len(savePath) > 0 {
		theSaver.SetLoadpath(&savePath)
		theSaver.SetSavepath(&savePath)
		log.Printf("set savepath:%v", savePath)
	}

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
		reqKey := v.Get("k")

		if len(reqKey) == 0 {
			log.Println("illegal message ignored. message=", string(message))
			ret.Data["list"] = nil
		} else {
			list := theSaver.GetList(&reqKey) //getListFromMc(mc, &reqKey)
			//tinyList := ToTinyOrbList(list)
			//ret.Data["list"] = tinyList
			ret.Data["list"] = ToFloatList(list)
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
			log.Println("write failed:", err)
			break
		}

		//time.Sleep(time.Millisecond * 100)
		//break
	}
	log.Println("closed:", c.RemoteAddr().String())
}

//func home(w http.ResponseWriter, r *http.Request) {
//	homeTemplate.Execute(w, "ws://"+r.Host+"/echo")
//}

// you must run it in the dir of index.html
func main() {
	var savePath = flag.String("savepath", "file://./", "where to save, support mc/file/redis\n\tlike: file://./filecache/\n\tor\n\tmc://127.0.0.1:11211")
	var webRoot = flag.String("webroot", ".", "where to serve web root, generally use the root of this project which include static files like index.html")
	flag.Parse()

	theSaver.SetSavepath(savePath)
	theSaver.SetLoadpath(savePath)

	log.SetFlags(log.Lshortfile | log.LstdFlags)

	http.Handle("/", http.FileServer(http.Dir(*webRoot)))
	http.HandleFunc("/echo", echo)
	http.HandleFunc("/orbs", handleOrbs)
	log.Println("server will start at", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
