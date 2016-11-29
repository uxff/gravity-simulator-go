package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"time"
	//"strings"
	//"github.com/bitly/go-simplejson"
	"github.com/bradfitz/gomemcache/memcache"
)

// 结构体中的变量必须大写才能被json输出 坑
type Orb struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	//Ax       float64 `json:"ax"`
	//Ay       float64 `json:"ay"`
	Vx       float64 `json:"vx"`
	Vy       float64 `json:"vy"`
	Dir      float64 `json:"dir"`
	Mass     float64 `json:"mass"`
	Size     float32 `json:"size"`
	LifeStep int     `json:"lifeStep"`
	Id       int     `json:"id"`
	//CalcTimes int     `json:"calcTimes"`
	//flag     int     `json:"flag"`
}
type Acc struct {
	Ax  float64
	Ay  float64
	A   float64
	Dir float64
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
		//o.Ax = 0.0
		//o.Ay = 0.0
		o.Dir = 0.0
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
	for i := 0; i < thelen; i++ {
		go mapHap[i].update(mapHap, c)
		//go updateOrb(&mapHap[i], mapHap)
		//fmt.Println("after the rand id=", mapHap[i].Id)
	}
	cCount := 0
	for {
		if cCount >= thelen {
			break
		}
		cCount += <-c
	}
	return cCount
}
func updateOrb(o *Orb, mapHap []Orb, c chan int) {
	//o.Mass += mapHap[0].Mass
	aAll := o.CalcGravityAll(mapHap)
	if o.LifeStep == 1 {
		//o.Ax = aAll.Ax
		//o.Ay = aAll.Ay
		o.Vx += aAll.Ax
		o.Vy += aAll.Ay
		o.X += o.Vx
		o.Y += o.Vy
	}
	//o.CalcTimes += 1
}
func (o *Orb) update(mapHap []Orb, c chan int) {
	//o.Mass += mapHap[0].Mass
	aAll := o.CalcGravityAll(mapHap)
	if o.LifeStep == 1 {
		//o.Ax = aAll.Ax
		//o.Ay = aAll.Ay
		o.Vx += aAll.Ax
		o.Vy += aAll.Ay
		o.X += o.Vx
		o.Y += o.Vy
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

		dist := calcDist(o.X, o.Y, target.X, target.Y)
		if dist < 1.0 {
			if o.Mass > target.Mass {
				o.Mass += target.Mass
				o.Vx = (target.Mass*target.Vx + o.Mass*o.Vx) / o.Mass
				o.Vy = (target.Mass*target.Vy + o.Mass*o.Vy) / o.Mass
				target.Mass = 0
				target.LifeStep = 2
			} else {
				target.Mass += target.Mass
				target.Vx = (target.Mass*target.Vx + o.Mass*o.Vx) / target.Mass
				target.Vy = (target.Mass*target.Vy + o.Mass*o.Vy) / target.Mass
				o.Mass = 0
				o.LifeStep = 2

			}
		} else {
			gTmp := o.CalcGravity(&oList[i], dist)
			gTmp.Ax = gTmp.A * math.Cos(gTmp.Dir)
			gTmp.Ay = gTmp.A * math.Sin(gTmp.Dir)
			gAll.Ax += gTmp.Ax
			gAll.Ay += gTmp.Ay
		}
	}

	return gAll
}
func (o *Orb) CalcGravity(target *Orb, dist float64) Acc {
	var a Acc
	if dist < 1.0 {
		return Acc{}
	}

	a.A = target.Mass / (dist * dist) * G
	a.Dir = math.Atan2((o.Y - target.Y), (o.X - target.X))
	return a
}
func calcDist(x1, y1, x2, y2 float64) float64 {
	return math.Sqrt((x1-x2)*(x1-x2) + (y1-y2)*(y1-y2))
}

func main() {

	num_orbs := MAX_PARTICLES
	num_times := FOR_TIMES
	// 使用2核心
	num_cores := 2
	var err error

	args := os.Args
	if args != nil && len(args) >= 2 {
		num_orbs, err = strconv.Atoi(os.Args[1])
	}
	if args != nil && len(args) >= 3 {
		num_times, err = strconv.Atoi(args[2])
	}

	if err != nil {
		fmt.Println("Args len", len(os.Args), "err:", err)
	}

	fmt.Println("useage: go_server.exe $num_orbs $num_times")
	fmt.Println("    eg: go_server.exe", num_orbs, num_times)

	// go 编译器自动选择最优核心数
	//runtime.GOMAXPROCS(num_cores)

	// 根据时间设置随机数种子
	rand.Seed(int64(time.Now().Nanosecond()))

	mapHap := initOrbs(num_orbs)
	//fmt.Println("after init mapHap=", mapHap)

	realTimes := 0
	//startTime := time.Now().Unix()
	startTimeNano := time.Now().UnixNano()

	for i := 0; i < num_times; i++ {
		realTimes += updateOrbs(mapHap)
	}
	//endTime := time.Now().Unix()
	endTimeNano := time.Now().UnixNano()
	timeUsed := float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Println("(USE GO, core:", num_cores, "/", runtime.NumCPU(), ") particles:", num_orbs, "for times:", num_times, "real:", realTimes, "use time:", timeUsed, "sec")

	mc := memcache.New("mc.lo:11211", "mc.lo:11211")

	if strList, err := json.Marshal(mapHap); err == nil {
		//fmt.Println("Marshal(mapHap) success: ", string(strList))
		theVal := strList //fmt.Sprintf(`{"code":0,"msg":"ok","data":{"list":%s}}`, strList)
		mc.Set(&memcache.Item{Key: "foo2", Value: []byte(theVal)})
	} else {
		fmt.Println("set foo2 error:", err)
	}
	if mcMapHap, err := mc.Get("foo2"); err == nil {
		fmt.Println("key=", mcMapHap.Key, " len(value)=", len(string(mcMapHap.Value)))
	} else {
		fmt.Println("get foo2 error:", err)
	}
	endTimeNano = time.Now().UnixNano()
	timeUsed = float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Println("all used time with mc->get/set:", timeUsed, "sec")
}
