package main

import (
	"encoding/json"
	"flag"
	"fmt"
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
	DoneList  []orbs.Orb
	MarkList  []byte
	WillCur   int
	DoneCount int
	Stage     int
}

func (this *CalcUnit) SetPrepList(list []orbs.Orb) {
	this.PrepList = list
	this.DoneList = list
	this.MarkList = make([]byte, len(list))
	this.WillCur = 0
	this.Stage = 1
}
func (this *CalcUnit) GetList() []orbs.Orb {
	return this.PrepList
}
func (this *CalcUnit) GetUncalcedId() (theIndex int, ok bool) {
	ok = false
	forTimes := 0

	for {
		if forTimes >= len(this.MarkList) {
			break
		}
		forTimes++
		if this.WillCur >= len(this.MarkList) {
			this.WillCur = 0
		}
		if this.MarkList[this.WillCur] == byte(0) {
			theIndex = this.WillCur
			ok = true
			this.WillCur++
			break
		}
		this.WillCur++
	}
	return theIndex, ok
}
func (this *CalcUnit) Reap(stage int, orb orbs.Orb, idx, crashedBy int) bool {
	//
	if stage != this.Stage {
		return false
	}
	if idx >= len(this.DoneList) {
		return false
	}
	if this.MarkList[idx] == byte(1) {
		log.Println("cannot repeat reap: stage,idx=", stage, idx)
		return false
	}
	if crashedBy >= 0 && crashedBy < len(this.DoneList) {
		//orb.CrashedBy = crashedBy
		//orb.SetCrashedBy(crashedBy)
		target := &this.DoneList[crashedBy]
		// 此处应该放在队列中，在所有stage升级的时候再处理crash事件
		// 或者判断target是否已经stage up,如果没有up，也不能操作target.mass,所以此方案不妥
		targetMassOld := target.Mass
		target.Mass += orb.Mass
		orb.Mass = 0
		target.Vx = (targetMassOld*target.Vx + orb.Mass*orb.Vx) / target.Mass
		target.Vy = (targetMassOld*target.Vy + orb.Mass*orb.Vy) / target.Mass
		target.Vz = (targetMassOld*target.Vz + orb.Mass*orb.Vz) / target.Mass
		target.Size++
	}
	this.DoneList[idx] = orb
	this.MarkList[idx] = byte(1)
	this.DoneCount++
	if this.DoneCount >= len(this.DoneList) {
		this.DoneCount = 0
		for i := 0; i < len(this.MarkList); i++ {
			this.MarkList[i] = byte(0)
		}
		this.PrepList = this.DoneList
		this.Stage++
	}
	return true
}

var allList = make(map[string]*CalcUnit)

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
		key := urlQuery.Get("k")
		ret.Data["cmd"] = cmd
		ret.Data["key"] = key

		if len(key) == 0 {
			ret.Msg = fmt.Sprintf("key empty")
			ret.Code = 0
		} else if len(cmd) == 0 {
			ret.Msg = fmt.Sprintf("cmd empty")
			ret.Code = 0
		} else {
			// 派发cmd
			switch cmd {
			case "orbs":
				var unit *CalcUnit
				if _, ok := allList[key]; ok {
					unit = allList[key]
				} else {
					unit = &CalcUnit{Key: key}
					unit.SetPrepList(saver.GetList(&key)) //getListFromMc(mc, &mcKey)
					allList[key] = unit
				}
				ret.Data["list"] = unit.PrepList
				ret.Data["stage"] = unit.Stage

				log.Println("get orbs done")
			case "taketask":
				calcNumVal := urlQuery.Get("calcnum")
				calcNum, _ := strconv.Atoi(calcNumVal)
				if calcNum == 0 {
					calcNum = 1
				}

				var unit *CalcUnit
				if _, ok := allList[key]; ok {
					unit = allList[key]
				} else {
					unit = &CalcUnit{Key: key}
					unit.SetPrepList(saver.GetList(&key)) //getListFromMc(mc, &mcKey)
					allList[key] = unit
				}
				var feedlist []int
				for i := 0; i < calcNum; i++ {
					curIndex, ok := unit.GetUncalcedId()
					if ok {
						feedlist = append(feedlist, curIndex)
					} else {
						ret.Msg = fmt.Sprintf("cannot get a appropriate orb")
						break
					}
				}
				ret.Data["feedlist"] = feedlist
				ret.Data["stage"] = unit.Stage

				log.Println("take a task done")
			case "recvorb":
				// request give o orb // compile ok
				var orb orbs.Orb
				orbStr := urlQuery.Get("o")
				stage, _ := strconv.Atoi(urlQuery.Get("stage"))
				theIdx := urlQuery.Get("idx")
				idx, _ := strconv.Atoi(theIdx)
				crashedBy, _ := strconv.Atoi(urlQuery.Get("crashedBy"))
				ret.Data["idx"] = idx
				ret.Data["stage"] = stage
				ret.Data["crashedBy"] = crashedBy

				jErr := json.Unmarshal([]byte(orbStr), &orb)
				if jErr != nil {
					ret.Msg = fmt.Sprintf("json unmarshal error:", jErr)
					ret.Code = 2
					break
				}
				unit, ok := allList[key]
				if !ok {
					ret.Msg = fmt.Sprintf("key not exist:%s", key)
					ret.Code = 2
					break
				}
				if stage <= 0 {
					ret.Msg = fmt.Sprintf("stage illegal:%v", stage)
					ret.Code = 2
					break
				} else if stage != unit.Stage {
					ret.Msg = fmt.Sprintf("stage(%d) not preg with unit.stage(%d):", stage, unit.Stage)
					ret.Code = 2
					break
				} else {
					unit.Reap(stage, orb, idx, crashedBy)
					time.Sleep(time.Second * 1)
					log.Println("save a orb: idx,crashedBy=", idx, crashedBy)
				}
			case "recvcrash":
				log.Println("recv a crash, this interface deleted")
			default:
				ret.Msg = "unknown cmd"
				ret.Code = 2
			}
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
