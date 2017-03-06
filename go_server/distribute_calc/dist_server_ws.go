package main

import (
	"encoding/json"
	"flag"
	//"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	orbs "../orbs"
	saverpkg "../saver"

	//"github.com/bradfitz/gomemcache/memcache"
	"github.com/gorilla/websocket"
)

// json返回值结构
type JsonRet struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data map[string]interface{} `json:"data"`
}
type CalcUnit struct {
	Key       string
	PrepList  []orbs.Orb
	DoneList  []byte
	WillCur   int
	DoneCount int
}

func (this *CalcUnit) SetPrepList(list []orbs.Orb) {
	this.PrepList = list
	this.DoneList = make([]byte, len(list))
	this.WillCur = 0
}
func (this *CalcUnit) GetUncalcedId() (theIndex int, ok bool) {
	ok = false
	forTimes := 0

	for {
		if forTimes >= len(this.DoneList) {
			break
		}
		forTimes++
		if this.WillCur >= len(this.DoneList) {
			this.WillCur = 0
		}
		if this.DoneList[this.WillCur] == byte(0) {
			theIndex = this.WillCur
			ok = true
			break
		}
		this.WillCur++
	}
	return theIndex, ok
}

var allList = make(map[string]CalcUnit)

var addr = flag.String("addr", "0.0.0.0:8082", "websocket server address of dist calc server")
var savePath = flag.String("savepath", "mc://127.0.0.1:11211", "where to save, support mc/file/redis\n\tlike: file://./filecache/")

//var savekey = flag.String("savekey", "thelist1", "key name to save, like key of memcache, or filename in save dir")
//var loadkey = flag.String("loadkey", "thelist1", "key name to load, like key of memcache, or filename in save dir")

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

func handleTask(w http.ResponseWriter, r *http.Request) {
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
		urlQuery, _ := url.ParseQuery(string(message))
		cmd := urlQuery.Get("cmd")
		ret.Data["cmd"] = cmd
		switch cmd {
		case "orbs":
			key := urlQuery.Get("k")

			if len(key) == 0 {
				log.Println("illegal message ignored. message=", string(message))
				ret.Data["alllist"] = nil
				break
			}
			var unit CalcUnit
			if _, ok := allList[key]; ok {
				unit = allList[key]
			} else {
				unit = CalcUnit{Key: key}
				unit.SetPrepList(saver.GetList(&key)) //getListFromMc(mc, &mcKey)

			}
			ret.Data["alllist"] = unit.PrepList

			log.Println("get orbs done")
		case "taketask":
			key := urlQuery.Get("k")
			calcNumVal := urlQuery.Get("calcnum")
			calcNum, _ := strconv.Atoi(calcNumVal)

			if len(key) == 0 {
				log.Println("illegal message ignored. message=", string(message))
				ret.Data["feedlist"] = make([]int, 0)
				break
			}
			var unit CalcUnit
			if _, ok := allList[key]; ok {
				unit = allList[key]
			} else {
				unit = CalcUnit{Key: key}
				unit.SetPrepList(saver.GetList(&key)) //getListFromMc(mc, &mcKey)
			}
			var feedlist []int
			for i := 0; i < calcNum; i++ {
				curIndex, ok := unit.GetUncalcedId()
				if ok {
					feedlist = append(feedlist, curIndex)
				} else {
					break
				}
			}
			ret.Data["feedlist"] = feedlist

			log.Println("take a task done")
		case "recvorb":
			// request give o orb // compile ok
			key := urlQuery.Get("k")
			var orb orbs.Orb
			if len(key) == 0 {
				log.Println("key not exist")
			}
			orbStr := urlQuery.Get("orb")
			theIdx := urlQuery.Get("idx")
			idx, _ := strconv.Atoi(theIdx)
			jErr := json.Unmarshal([]byte(orbStr), &orb)
			if jErr != nil {
				log.Println("fuck json error:", jErr)
			}
			unit, ok := allList[key]
			if !ok {
				log.Println("key not exist:", key)
			}
			unit.DoneList[idx] = 1
			unit.DoneCount++
			unit.PrepList[idx] = orb
			time.Sleep(time.Second * 1)
			log.Println("save a orb")
		case "recvcrash":
			log.Println("recv a crash")
		default:
			ret.Msg = "unknown cmd"
			ret.Code = 2
		}

		retStr, errJson := json.Marshal(ret)
		if errJson != nil {
			log.Println("json.Marshal error:", errJson)
			break
		}

		err = c.WriteMessage(mt, retStr)
		if err != nil {
			log.Println("write failed:", err)
			break
		}

	}
	log.Println("closed:", c.LocalAddr().String())
}

//func home(w http.ResponseWriter, r *http.Request) {
//	homeTemplate.Execute(w, "ws://"+r.Host+"/echo")
//}

func main() {
	flag.Parse()

	saver.SetSavepath(savePath)

	log.SetFlags(log.Lshortfile | log.LstdFlags)

	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/echo", echo)
	http.HandleFunc("/orbs", handleTask)
	log.Println("server will start at", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
