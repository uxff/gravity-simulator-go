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

type InitConfig struct {
	Mass    float64
	Wide    float64
	Velo    float64
	Eternal float64
}

const G = 0.000021
const MAX_PARTICLES = 100
const FOR_TIMES = 10000
const VELO = 0.005

// 最小距离值
const MIN_CRITICAL_DIST = 3.0

var maxVeloX, maxVeloY, maxVeloZ, maxAccX, maxAccY, maxAccZ float64 = 0, 0, 0, 0, 0, 0

//var nStep int

func initOrbs(num int, config *InitConfig) []Orb {
	oList := make([]Orb, num)

	if config.Eternal != 0.0 {
		num -= 1
	}

	for i := 0; i < num; i++ {
		o := &oList[i]

		o.X, o.Y = (1.0-rand.Float64())*config.Wide, (1.0-rand.Float64())*config.Wide
		o.Z = (1.0 - rand.Float64()) * config.Wide
		o.Vx = (rand.Float64() - 1.0) * config.Velo * 2.0
		o.Vy = (rand.Float64() - 1.0) * config.Velo * 2.0
		o.Vz = (rand.Float64())*config.Velo*2.0 - config.Velo
		//o.Dir = 0.0
		o.Size = 1 //float32(math.Sqrt(o.X * o.Y))
		o.Mass = rand.Float64() * config.Mass
		//o.Id = rand.Int()
		o.Id = i
		o.LifeStep = 1
		//fmt.Println("the rand id=", o.Id)
	}
	if config.Eternal != 0.0 {
		eternalOrb := &oList[num]
		//eternalOrb.X = 0,
		eternalOrb.Mass = config.Eternal
		eternalOrb.Id = num //rand.Int()
	}
	return oList
}

func updateOrbs(oList []Orb, nStep int) int {
	thelen := len(oList)
	c := make(chan int)
	cCount := 0
	//fmt.Println("will start times(", nStep, ") updateOrbs()")
	for i := 0; i < thelen; i++ {
		//fmt.Println("will start nStep(", nStep, ") orb[", i, "].update()")
		go oList[i].update(oList, c, nStep)
		//go updateOrb(&oList[i], oList, c) // you can run this not with go
	}
	//cCount += 1
	for {
		if cCount >= thelen {
			break
		}
		cCount += <-c
	}
	//fmt.Println("will end nStep(", nStep, ") updateOrbs()")
	return cCount * cCount
}
func (o *Orb) update(oList []Orb, c chan int, nStep int) {
	//o.Mass += oList[0].Mass
	//fmt.Println("in nStep(", nStep, ") orb[", o.Id, "].update() before calc")
	aAll := o.CalcGravityAll(oList)
	if o.LifeStep == 1 {
		o.Vx += aAll.Ax
		o.Vy += aAll.Ay
		o.Vz += aAll.Az
		o.X += o.Vx
		o.Y += o.Vy
		o.Z += o.Vz
		if maxVeloX < o.Vx {
			maxVeloX = o.Vx
		}
		if maxVeloY < o.Vy {
			maxVeloY = o.Vy
		}
		if maxVeloZ < o.Vz {
			maxVeloZ = o.Vz
		}
		if maxAccX < aAll.Ax {
			maxAccX = aAll.Ax
		}
		if maxAccY < aAll.Ay {
			maxAccY = aAll.Ay
		}
		if maxAccZ < aAll.Az {
			maxAccZ = aAll.Az
		}
	}
	//o.CalcTimes += 1
	//fmt.Println("in nStep(", nStep, ") orb[", o.Id, "].update() before c<-")
	c <- 1 //len(oList)
	//fmt.Println("in nStep(", nStep, ") orb[", o.Id, "].update() after c<-")
}
func (o *Orb) CalcGravityAll(oList []Orb) Acc {
	var gAll Acc
	for i := 0; i < len(oList); i++ {
		//c <- 1
		var isCrash, isTooNearly, isVertDistBigger, isSpanOn bool = false, false, false, false
		target := &oList[i]
		if target.Id == o.Id || target.LifeStep != 1 || o.LifeStep != 1 {
			//fmt.Println("orb cannot act on self, or life over")
			continue
		}

		dist := o.calcDist(target)

		isTooNearly = dist*dist < MIN_CRITICAL_DIST*MIN_CRITICAL_DIST
		// 如果半径小，则相交的两个角必须是锐角，才能确定是碰撞
		verticalX, verticalY, verticalZ := o.calcVertiDot(target)
		isVertDistBigger = ((verticalX-target.X)*(verticalX-target.X) + (verticalY-target.Y)*(verticalY-target.Y) + (verticalZ-target.Z)*(verticalZ-target.Z)) < MIN_CRITICAL_DIST*MIN_CRITICAL_DIST

		isSpanOn = ((o.X-o.Vx-target.X)*(o.X-o.Vx-target.X)+(o.Y-o.Vy-target.Y)*(o.Y-o.Vy-target.Y)+(o.Z-o.Vz-target.Z)*(o.Z-o.Vz-target.Z)+dist*dist) > (o.Vx*o.Vx+o.Vy*o.Vy+o.Vz*o.Vz) && ((o.X-target.X)*(o.X-target.X)+(o.Y-target.Y)*(o.Y-target.Y)+(o.Z-target.Z)*(o.Z-target.Z)+dist*dist) > (o.Vx*o.Vx+o.Vy*o.Vy+o.Vz*o.Vz)
		// 如果垂心距离target比临街半径大 则不相交
		// da^2 + do^2 > db^2 && db^2 + do^2 > da^2
		isCrash = isTooNearly || isVertDistBigger && isSpanOn
		if isCrash {

			// 碰撞机制 非弹性碰撞 动量守恒 m1v1+m2v2=(m1+m2)v
			if o.Mass > target.Mass {
				// 碰撞后速度 v = (m1v1+m2v2)/(m1+m2)
				o.Mass += target.Mass
				o.Vx = (target.Mass*target.Vx + o.Mass*o.Vx) / o.Mass
				o.Vy = (target.Mass*target.Vy + o.Mass*o.Vy) / o.Mass
				o.Vz = (target.Mass*target.Vz + o.Mass*o.Vz) / o.Mass
				o.Size += 1
				target.Mass = 0
				target.LifeStep = 2
			} else {
				target.Mass += target.Mass
				target.Vx = (target.Mass*target.Vx + o.Mass*o.Vx) / target.Mass
				target.Vy = (target.Mass*target.Vy + o.Mass*o.Vy) / target.Mass
				target.Vz = (target.Mass*target.Vz + o.Mass*o.Vz) / target.Mass
				target.Size += 1
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
func (o *Orb) calcVertiDot(target *Orb) (vx, vy, vz float64) {
	//k := (o.Y - target.Y)/(o.X - target.X)
	//k := o.Vx / o.Vy
	//3d
	// k = -((x1-x0)(x2-x1)+(y2-y1)(y1-y0)+(z2-z1)(z1-z0))/(x2-x1)^2+(y2-y1)^2+(z2-z1)^2
	// xn=k(x2-x1)+x1 yn=k(y2-y1)+y1 zn=k(z2-z1)
	var x0, x1, x2, y0, y1, y2, z0, z1, z2 float64 = target.X, o.X, o.X - o.Vx, target.Y, o.Y, o.Y - o.Vy, target.Z, o.Z, o.Z - o.Vz
	k := -((x1-x0)*(x2-x1) + (y2-y1)*(y1-y0) + (z2-z1)*(z1-z0)) / ((x2-x1)*(x2-x1) + (y2-y1)*(y2-y1) + (z2-z1)*(z2-z1))
	vx = k*(x2-x1) + x1
	vy = k*(y2-y1) + y1
	vz = k*(z2-z1) + z1

	// x=(t.y+t.x/k+kx2-y2)/(k+1/k)
	// 直线(o.x-o.vx, o.y-o.vy),(o.x, o.y)的直线为
	// y=kx+y2-kx2
	// 点target到上面直线的垂线为
	// y=-1/k x + t.y + t.x/k
	//vx = (target.Y + target.X/k + k*o.X - o.Y) / (k + 1/k)
	//vy = k*vx + o.Y - k*o.X
	return vx, vy, vz
}

// 从数据库获取orbList
func getListFromMc(mc *memcache.Client, mcKey *string) (oList []Orb, v []byte) {
	var orbListStr string
	if orbListStrVal, err := mc.Get(*mcKey); err == nil {
		v = orbListStrVal.Value
		orbListStr = string(orbListStrVal.Value)
		err := json.Unmarshal(orbListStrVal.Value, &oList)
		fmt.Println("mc.get len(val)=", len(orbListStr), "after unmarshal, len=", len(oList), "json err=", err)
	} else {
		fmt.Println("mc.get", *mcKey, "error:", err)
	}
	return oList, v
}

// 将orbList存到数据库
func saveListToMc(mc *memcache.Client, mcKey *string, oList []Orb) {
	if strList, err := json.Marshal(oList); err == nil {
		theVal := strList //fmt.Sprintf(`{"code":0,"msg":"ok","data":{"list":%s}}`, strList)
		errMc := mc.Set(&memcache.Item{Key: *mcKey, Value: []byte(theVal)})
		if errMc != nil {
			fmt.Println("save failed:", errMc)
		} else {
			fmt.Println("save success: len=", len(oList))
		}
	} else {
		fmt.Println("set", mcKey, "error:", err)
	}
}

// 清理orbList中的垃圾
func clearOrbList(oList []Orb) []Orb {
	for i := 0; i < len(oList); i++ {
		if oList[i].LifeStep != 1 {
			oList = append(oList[:i], oList[i+1:]...)
			i--
		}
	}
	return oList
}

func main() {
	num_orbs := MAX_PARTICLES
	num_times := FOR_TIMES
	//doInit := false
	var eternal float64
	var mcHost, mcKey string
	var numCpu int

	flag.IntVar(&num_orbs, "init_orbs", 0, "how many orbs init, do init when its value >1")
	flag.IntVar(&num_times, "num_times", 100, "how many times calc")
	flag.Float64Var(&eternal, "eternal", 15000.0, "the mass of eternal, 0 means no eternal")
	flag.StringVar(&mcHost, "mc_host", "mc.lo:11211", "memcache server")
	flag.StringVar(&mcKey, "mc_key", "mcasync2", "key name save into memcache")
	var doShowList = flag.Bool("show_list", false, "show orb list and exit")
	var configMass = flag.Float64("config-mass", 10.0, "the mass of orbs")
	var configWide = flag.Float64("config-wide", 1000.0, "the wide of orbs")
	var configVelo = flag.Float64("config-velo", 0.005, "the velo of orbs")
	var configCpu = flag.Int("config-cpu", 0, "how many cpu u want use, 0=all")

	// flags 读取参数，必须要调用 flag.Parse()
	flag.Parse()

	if *configCpu > 0 {
		numCpu = *configCpu
	} else {
		numCpu = runtime.NumCPU() - 1
	}
	runtime.GOMAXPROCS(numCpu)
	//fmt.Println("useage: go_server.exe $num_orbs $num_times")
	fmt.Println("    eg: go_server.exe -num_orbs", num_orbs, "-num_times", num_times)

	var oList []Orb
	var mcVal []byte
	mc := memcache.New(mcHost) //New("mc.lo:11211", "mc.lo:11211")

	// 根据时间设置随机数种子
	rand.Seed(int64(time.Now().Nanosecond()))

	if num_orbs > 0 {
		initConfig := InitConfig{*configMass, *configWide, *configVelo, eternal}
		oList = initOrbs(num_orbs, &initConfig)
		//fmt.Println("after init oList=", oList)
	} else {
		oList, mcVal = getListFromMc(mc, &mcKey)
	}
	if *doShowList {
		fmt.Println(string(mcVal))
		fmt.Println(oList)
		return
	}
	num_orbs = len(oList)

	realTimes, perTimes, tmpTimes := 0, 0, 0
	//startTime := time.Now().Unix()
	startTimeNano := time.Now().UnixNano()

	for i := 0; i < num_times; i++ {
		perTimes = updateOrbs(oList, i)
		realTimes += perTimes
		//fmt.Printf("in main oList=%p\n", oList)//slice地址一直是一样的，除非append

		//nStep = i
		//if (i*10+1)%(num_times+1) == 1 {
		//}
		tmpTimes += perTimes
		if tmpTimes > 100000 {
			saveListToMc(mc, &mcKey, oList)
			oList = clearOrbList(oList)
			tmpTimes = 0
		}
	}

	oList = clearOrbList(oList)

	//endTime := time.Now().Unix()
	endTimeNano := time.Now().UnixNano()
	timeUsed := float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Println("(core:", numCpu, ") orbs:", num_orbs, len(oList), "times:", num_times, "real:", realTimes, "use time:", timeUsed, "sec", "CPS:", float64(realTimes)/timeUsed)
	fmt.Println("maxVelo=", maxVeloX, maxVeloY, maxVeloZ, "maxAcc=", maxAccX, maxAccY, maxAccZ)

	saveListToMc(mc, &mcKey, oList)

	endTimeNano = time.Now().UnixNano()
	timeUsed = float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Println("all used time with mc->get/set:", timeUsed, "sec")
}
