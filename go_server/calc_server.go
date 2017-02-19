package main

import (
	//"bytes"
	//"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	//"os"
	"runtime"
	//"strconv"
	"time"
	//"strings"
)

// 结构体中的变量必须大写才能被json输出 坑
// 天体结构体声明
type Orb struct {
	X        float64 `json:"x"`        // 坐标x
	Y        float64 `json:"y"`        // 坐标y
	Z        float64 `json:"z"`        // 坐标z
	Vx       float64 `json:"vx"`       // 速度x
	Vy       float64 `json:"vy"`       // 速度y
	Vz       float64 `json:"vz"`       // 速度z
	Mass     float64 `json:"mass"`     // 质量
	Size     int     `json:"size"`     // 大小，用于计算吞并的天体数量
	LifeStep int     `json:"lifeStep"` // 用于标记是否已爆炸 1=正常 2=已爆炸
	Id       int     `json:"id"`
	//CalcTimes int     `json:"calcTimes"`
}

// 加速度结构体
type Acc struct {
	Ax float64
	Ay float64
	Az float64
	A  float64
}

// 配置
type InitConfig struct {
	Mass    float64
	Wide    float64
	Velo    float64
	Eternal float64
}

// 万有引力常数
const G = 0.000021

// 默认最大天体数量
const MAX_PARTICLES = 100

// 默认计算步数
const FOR_TIMES = 10000

// 最小天体距离值 两天体距离小于此值了会相撞
const MIN_CRITICAL_DIST = 1.0

// 监控速度和加速度
var maxVeloX, maxVeloY, maxVeloZ, maxAccX, maxAccY, maxAccZ, maxMass float64 = 0, 0, 0, 0, 0, 0, 0
var maxMassId int = 0

var saver = Saver{}

// 初始化天体位置，质量，加速度 在一片区域随机分布
func initOrbs(num int, config *InitConfig) []Orb {
	oList := make([]Orb, num)

	for i := 0; i < num; i++ {
		o := &oList[i]

		o.X, o.Y, o.Z = (0.5-rand.Float64())*config.Wide, (0.5-rand.Float64())*config.Wide, (0.5-rand.Float64())*config.Wide
		o.Vx = (rand.Float64() - 0.5) * config.Velo * 2.0
		o.Vy = (rand.Float64() - 0.5) * config.Velo * 2.0
		o.Vz = (rand.Float64() - 0.5) * config.Velo * 2.0
		o.Size = 1
		o.Mass = rand.Float64() * config.Mass
		o.Id = i // rand.Int()
		o.LifeStep = 1
	}
	// 如果配置了恒星，将最后一个设置为恒星
	if config.Eternal != 0.0 {
		eternalId := num - 1
		eternalOrb := &oList[eternalId]
		eternalOrb.Mass = config.Eternal
		eternalOrb.Id = eternalId //rand.Int()
		eternalOrb.X, eternalOrb.Y, eternalOrb.Z = 0, 0, 0
	}
	return oList
}

// 所有天体运动一次
func updateOrbs(oList []Orb, nStep int) int {
	thelen := len(oList)
	c := make(chan int)
	cCount := 0
	for i := 0; i < thelen; i++ {
		go oList[i].update(oList, c, nStep)
	}
	for {
		if cCount >= thelen {
			break
		}
		cCount += <-c
		//cCount += 1
	}
	// 以下方法运行时报错
	//	for cval := range c {
	//		cCount += cval
	//	}
	return cCount * cCount
}

// 天体运动一次
func (o *Orb) update(oList []Orb, c chan int, nStep int) {
	aAll := o.CalcGravityAll(oList)
	if o.LifeStep == 1 {
		o.Vx += aAll.Ax
		o.Vy += aAll.Ay
		o.Vz += aAll.Az
		o.X += o.Vx
		o.Y += o.Vy
		o.Z += o.Vz
		// 监控速度和加速度
		if maxVeloX < math.Abs(o.Vx) {
			maxVeloX = math.Abs(o.Vx)
		}
		if maxVeloY < math.Abs(o.Vy) {
			maxVeloY = math.Abs(o.Vy)
		}
		if maxVeloZ < math.Abs(o.Vz) {
			maxVeloZ = math.Abs(o.Vz)
		}
		if maxAccX < math.Abs(aAll.Ax) {
			maxAccX = math.Abs(aAll.Ax)
		}
		if maxAccY < math.Abs(aAll.Ay) {
			maxAccY = math.Abs(aAll.Ay)
		}
		if maxAccZ < math.Abs(aAll.Az) {
			maxAccZ = math.Abs(aAll.Az)
		}
		if maxMass < o.Mass {
			maxMass = o.Mass
			maxMassId = o.Id
		}
	}
	//o.CalcTimes += 1
	c <- 1 //len(oList)
}

// 计算天体受到的总体引力
func (o *Orb) CalcGravityAll(oList []Orb) Acc {
	var gAll Acc
	for i := 0; i < len(oList); i++ {
		//c <- 1
		target := &oList[i]
		if target.Id == o.Id || target.LifeStep != 1 || o.LifeStep != 1 || o.Mass == 0 || target.Mass == 0 {
			continue
		}

		dist := o.CalcDist(target)

		// 距离太近，被撞
		isTooNearly := dist*dist < MIN_CRITICAL_DIST*MIN_CRITICAL_DIST
		// 速度太快，被撕裂
		isTaRipped := dist*dist < (target.Vx*target.Vx+target.Vy*target.Vy+target.Vz*target.Vz)*10

		if isTooNearly || isTaRipped {

			// 碰撞机制 非弹性碰撞 动量守恒 m1v1+m2v2=(m1+m2)v
			if o.Mass > target.Mass {
				//fmt.Println(o.Id, "crashed", target.Id, "isTooNearly", isTooNearly, "me=", o, "ta=", target)
				// 碰撞后速度 v = (m1v1+m2v2)/(m1+m2)
				o.Mass += target.Mass
				o.Vx = (target.Mass*target.Vx + o.Mass*o.Vx) / o.Mass
				o.Vy = (target.Mass*target.Vy + o.Mass*o.Vy) / o.Mass
				o.Vz = (target.Mass*target.Vz + o.Mass*o.Vz) / o.Mass
				o.Size += 1
				target.Mass = 0
				target.LifeStep = 2
			} else {
				//fmt.Println(o.Id, "crashed by", target.Id, "isTooNearly", isTooNearly, "me=", o, "ta=", target)
				target.Mass += target.Mass
				target.Vx = (target.Mass*target.Vx + o.Mass*o.Vx) / target.Mass
				target.Vy = (target.Mass*target.Vy + o.Mass*o.Vy) / target.Mass
				target.Vz = (target.Mass*target.Vz + o.Mass*o.Vz) / target.Mass
				target.Size += 1
				o.Mass = 0
				o.LifeStep = 2
			}
		} else {
			// 作用正常，累计计算受到的所有的万有引力
			gTmp := o.CalcGravity(&oList[i], dist)
			gAll.Ax += gTmp.Ax
			gAll.Ay += gTmp.Ay
			gAll.Az += gTmp.Az
		}
	}

	return gAll
}

// 计算天体与目标的引力
func (o *Orb) CalcGravity(target *Orb, dist float64) Acc {
	var a Acc
	// 万有引力公式
	a.A = target.Mass / (dist * dist) * G
	a.Ax = -a.A * (o.X - target.X) / dist //a.A * math.Cos(a.Dir)
	a.Ay = -a.A * (o.Y - target.Y) / dist //a.A * math.Sin(a.Dir)
	a.Az = -a.A * (o.Z - target.Z) / dist //a.A * math.Sin(a.Dir)
	return a
}

// 计算距离
func (o *Orb) CalcDist(target *Orb) float64 {
	return math.Sqrt((o.X-target.X)*(o.X-target.X) + (o.Y-target.Y)*(o.Y-target.Y) + (o.Z-target.Z)*(o.Z-target.Z))
}

// 从数据库获取orbList
func getList(key *string) (oList []Orb) {
	return saver.GetHandler().LoadList(key)
}

// 将orbList存到数据库
func saveList(key *string, oList []Orb) {
	saver.GetHandler().SaveList(key, oList)
}

// 清理orbList中的垃圾
func clearOrbList(oList []Orb) []Orb {
	var alive int = len(oList)
	for i := 0; i < len(oList); i++ {
		if oList[i].LifeStep != 1 {
			oList = append(oList[:i], oList[i+1:]...)
			i--
			alive--
		}
	}
	//fmt.Println("when clear alive=", alive)
	return oList
}

func main() {
	num_orbs := MAX_PARTICLES
	num_times := FOR_TIMES
	var eternal float64
	var mcHost, mcKey string
	var numCpu int

	flag.IntVar(&num_orbs, "init-orbs", 0, "how many orbs init, do init when its value >1")
	flag.IntVar(&num_times, "calc-times", 100, "how many times calc")
	flag.Float64Var(&eternal, "eternal", 15000.0, "the mass of eternal, 0 means no eternal")
	flag.StringVar(&mcHost, "mchost", "127.0.0.1:11211", "memcache server")
	flag.StringVar(&mcKey, "savekey", "thelist1", "key name save into memcache")
	var doShowList = flag.Bool("showlist", false, "show orb list and exit")
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

	var oList []Orb

	var htype int = 2
	//saverConf := map[string]string{"dir": "./go_server/filecache/"}
	saverConf := map[string]string{"host": "mc.lo:11211"}
	saver.SetHandler(htype, saverConf)

	// 根据时间设置随机数种子
	rand.Seed(int64(time.Now().Nanosecond()))

	if num_orbs > 0 {
		initConfig := InitConfig{*configMass, *configWide, *configVelo, eternal}
		oList = initOrbs(num_orbs, &initConfig)
	} else {
		oList = getList(&mcKey)
	}
	if *doShowList {
		fmt.Println(oList)
		return
	}
	num_orbs = len(oList)

	realTimes, perTimes, tmpTimes, saveTimes := 0, 0, 0, 0
	startTimeNano := time.Now().UnixNano()

	for i := 0; i < num_times; i++ {
		perTimes = updateOrbs(oList, i)
		realTimes += perTimes

		tmpTimes += perTimes
		if tmpTimes > 5000000 {
			saveList(&mcKey, oList)
			oList = clearOrbList(oList)
			tmpTimes = 0
			saveTimes++
		}
	}

	oList = clearOrbList(oList)
	//fmt.Println("when clear oList=", oList)

	endTimeNano := time.Now().UnixNano()
	timeUsed := float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Println("core:", numCpu, " orbs:", num_orbs, len(oList), "times:", num_times, "real:", realTimes, "use time:", timeUsed, "sec", "CPS:", float64(realTimes)/timeUsed)
	fmt.Println("maxVelo=", maxVeloX, maxVeloY, maxVeloZ, "maxAcc=", maxAccX, maxAccY, maxAccZ, "maxMass", maxMassId, maxMass)

	saveList(&mcKey, oList)
	saveTimes++

	endTimeNano = time.Now().UnixNano()
	timeUsed = float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Println("all used time with save:", timeUsed, "sec, saveTimes=", saveTimes, "save per sec=", float64(saveTimes)/timeUsed)
}
