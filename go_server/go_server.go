package main

import (
	//"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	//"os"
	"runtime"
	//"strconv"
	"time"
	//"strings"
	//"github.com/bitly/go-simplejson"
	"github.com/bradfitz/gomemcache/memcache"
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
	Id       int     `json:"id"`
	//CalcTimes int     `json:"calcTimes"`
	//flag     int     `json:"flag"`
}
type Acc struct {
	Ax float64
	Ay float64
	Az float64
	A  float64
	//Dir float64
}

const G = 0.000021
const MAX_PARTICLES = 100
const FOR_TIMES = 10000
const VELO = 0.005

func initOrbs(num int) []Orb {
	mapHap := make([]Orb, num)
	for i := 0; i < num; i++ {
		o := &mapHap[i]

		o.X, o.Y = rand.Float64()*1000, rand.Float64()*1000
		o.Z = rand.Float64() * 1000
		//o.Ax = 0.0
		//o.Ay = 0.0
		//o.Dir = 0.0
		//o.Size = float32(math.Sqrt(o.X * o.Y))
		o.Mass = rand.Float64() * 2.0
		o.Id = rand.Int()
		o.LifeStep = 1
		//fmt.Println("the rand id=", o.Id)
	}
	return mapHap
}

func updateOrbs(mapHap []Orb) int {
	thelen := len(mapHap)
	c := make(chan int)
	cCount := 0
	for i := 0; i < thelen; i++ {
		go mapHap[i].update(mapHap, c)
		//go updateOrb(&mapHap[i], mapHap, c) // you can run this not with go
	}
	//cCount += 1
	defer func() {
		for {
			if cCount >= thelen {
				break
			}
			cCount += <-c
		}
	}()
	return cCount
}
func (o *Orb) update(mapHap []Orb, c chan int) {
	//o.Mass += mapHap[0].Mass
	aAll := o.CalcGravityAll(mapHap)
	if o.LifeStep == 1 {
		o.Vx += aAll.Ax
		o.Vy += aAll.Ay
		o.Vz += aAll.Az
		o.X += o.Vx
		o.Y += o.Vy
		o.Z += o.Vz
	}
	//o.CalcTimes += 1
	c <- 1
}
func (o *Orb) CalcGravityAll(oList []Orb) Acc {
	var gAll Acc
	for i := 0; i < len(oList); i++ {
		//c <- 1
		target := &oList[i]
		if target.Id == o.Id || target.LifeStep != 1 || o.LifeStep != 1 {
			//fmt.Println("orb cannot act on self, or life over")
			continue
		}

		dist := o.calcDist(target)
		if dist < 2.0 {
			// 碰撞机制 非弹性碰撞 动量守恒 m1v1+m2v2=(m1+m2)v
			if o.Mass > target.Mass {
				// 碰撞后速度 v = (m1v1+m2v2)/(m1+m2)
				o.Mass += target.Mass
				o.Vx = (target.Mass*target.Vx + o.Mass*o.Vx) / o.Mass
				o.Vy = (target.Mass*target.Vy + o.Mass*o.Vy) / o.Mass
				o.Vz = (target.Mass*target.Vz + o.Mass*o.Vz) / o.Mass
				target.Mass = 0
				target.LifeStep = 2
			} else {
				target.Mass += target.Mass
				target.Vx = (target.Mass*target.Vx + o.Mass*o.Vx) / target.Mass
				target.Vy = (target.Mass*target.Vy + o.Mass*o.Vy) / target.Mass
				target.Vz = (target.Mass*target.Vz + o.Mass*o.Vz) / target.Mass
				o.Mass = 0
				o.LifeStep = 2
			}
		} else {
			gTmp := o.CalcGravity(&oList[i], dist)
			gAll.Ax += gTmp.Ax
			gAll.Ay += gTmp.Ay
			gAll.Az += gTmp.Az
		}
	}

	return gAll
}
func (o *Orb) CalcGravity(target *Orb, dist float64) Acc {
	var a Acc
	// 万有引力公式
	a.A = target.Mass / (dist * dist) * G
	//a.Dir = math.Atan2((o.Y - target.Y), (o.X - target.X))
	a.Ax = -a.A * (o.X - target.X) / dist //a.A * math.Cos(a.Dir)
	a.Ay = -a.A * (o.Y - target.Y) / dist //a.A * math.Sin(a.Dir)
	a.Az = -a.A * (o.Z - target.Z) / dist //a.A * math.Sin(a.Dir)
	return a
}
func (o *Orb) calcDist(target *Orb) float64 {
	return math.Sqrt((o.X-target.X)*(o.X-target.X) + (o.Y-target.Y)*(o.Y-target.Y) + (o.Z-target.Z)*(o.Z-target.Z))
}

// 从数据库获取orbList
func getListFromMc(mc *memcache.Client, mcKey *string) (mapHap []Orb) {
	var orbListStr string
	if orbListStrVal, err := mc.Get(*mcKey); err == nil {
		orbListStr = string(orbListStrVal.Value)
		err := json.Unmarshal(orbListStrVal.Value, &mapHap)
		fmt.Println("len(val)=", len(orbListStr), "after unmarshal, len=", len(mapHap), "err:", err)
	} else {
		fmt.Println("mc get", *mcKey, "error:", err)
	}
	return mapHap
}

// 将orbList存到数据库
func saveListToMc(mc *memcache.Client, mcKey *string, mapHap []Orb) {
	if strList, err := json.Marshal(mapHap); err == nil {
		theVal := strList //fmt.Sprintf(`{"code":0,"msg":"ok","data":{"list":%s}}`, strList)
		mc.Set(&memcache.Item{Key: *mcKey, Value: []byte(theVal)})
	} else {
		fmt.Println("set", mcKey, "error:", err)
	}
}

func main() {
	num_orbs := MAX_PARTICLES
	num_times := FOR_TIMES
	doInit := false

	var mcHost, mcKey string
	flag.IntVar(&num_orbs, "num_orbs", 20, "how many orbs init")
	flag.IntVar(&num_times, "num_times", 100, "how many times calc")
	flag.StringVar(&mcHost, "mc_host", "mc.lo:11211", "memcache server")
	flag.StringVar(&mcKey, "mc_key", "mcasync2", "key name save into memcache")
	flag.BoolVar(&doInit, "doinit", false, "do init orb list and do other")
	// flags 读取参数，必须要调用 flag.Parse()
	flag.Parse()

	//fmt.Println("useage: go_server.exe $num_orbs $num_times")
	fmt.Println("    eg: go_server.exe -num_orbs", num_orbs, "-num_times", num_times)

	var mapHap []Orb
	//gobDecoder := gob.NewDecoder()
	mc := memcache.New(mcHost) //New("mc.lo:11211", "mc.lo:11211")

	// 根据时间设置随机数种子
	rand.Seed(int64(time.Now().Nanosecond()))

	if doInit {
		mapHap = initOrbs(num_orbs)
		//fmt.Println("after init mapHap=", mapHap)
	} else {
		mapHap = getListFromMc(mc, &mcKey)
	}
	num_orbs = len(mapHap)

	realTimes := 0
	//startTime := time.Now().Unix()
	startTimeNano := time.Now().UnixNano()

	for i := 0; i < num_times; i++ {
		realTimes += updateOrbs(mapHap)
		if ((num_orbs * num_orbs * i) % 100000) == 99999 {
			saveListToMc(mc, &mcKey, mapHap)
		}
	}
	//endTime := time.Now().Unix()
	endTimeNano := time.Now().UnixNano()
	timeUsed := float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Println("(USE GO, core:", runtime.NumCPU(), ") particles:", num_orbs, "for times:", num_times, "real:", realTimes, "use time:", timeUsed, "sec")

	saveListToMc(mc, &mcKey, mapHap)

	endTimeNano = time.Now().UnixNano()
	timeUsed = float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Println("all used time with mc->get/set:", timeUsed, "sec")
}
