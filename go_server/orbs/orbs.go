/*
	天体及天体计算框架
*/
package orbs

import (
	"log"
	"math"
	"math/rand"
)

// 天体结构体声明
type Orb struct {
	X    float64 `json:"x"`  // 坐标x
	Y    float64 `json:"y"`  // 坐标y
	Z    float64 `json:"z"`  // 坐标z
	Vx   float64 `json:"vx"` // 速度x
	Vy   float64 `json:"vy"` // 速度y
	Vz   float64 `json:"vz"` // 速度z
	Mass float64 `json:"m"`  // 质量
	Size int     `json:"sz"` // 大小，用于计算吞并的天体数量
	Stat int     `json:"st"` // 用于标记是否已爆炸 1=正常 2=已爆炸
	Id   int     `json:"id"`
	idx  int
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
	Style   int // 分布方式 1=立方体 2=圆盘圆柱 3=球形
}

// 万有引力常数
const G = 0.000021

// 最小天体距离值 两天体距离小于此值了会相撞
const MIN_CRITICAL_DIST = 1.0

// 监控速度和加速度
var maxVeloX, maxVeloY, maxVeloZ, maxAccX, maxAccY, maxAccZ, maxMass, allMass, allWC float64 = 0, 0, 0, 0, 0, 0, 0, 0, 0
var maxMassId, clearTimes int = 0, 0

var c = make(chan int, 10)
var nCount, nCrashed int = 0, 0

type crashEvent struct {
	idx        int
	crashedIdx int
}

var crashList = make(chan crashEvent, 10)

// 初始化天体位置，质量，加速度 在一片区域随机分布
func InitOrbs(num int, config *InitConfig) []Orb {
	oList := make([]Orb, num)

	switch config.Style {
	case 0:
		fallthrough
	case 1: //立方体
		for i := 0; i < num; i++ {
			o := &oList[i]
			o.X, o.Y, o.Z = (0.5-rand.Float64())*config.Wide, (0.5-rand.Float64())*config.Wide, (0.5-rand.Float64())*config.Wide
			o.Vx = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Vy = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Vz = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Size = 1
			o.Mass = rand.Float64() * config.Mass
			o.Id = i // rand.Int()
			o.Stat = 1
			allMass += o.Mass
		}
	case 2: //圆盘 随机选经度 随机选半径 随机选高低 刻意降低垂直于柱面的速度
		for i := 0; i < num; i++ {
			o := &oList[i]
			long := rand.Float64() * math.Pi * 2
			radius := rand.Float64() * config.Wide / 2.0
			high := (0.5 - rand.Float64()) * config.Wide
			o.X, o.Y = math.Cos(long)*radius, math.Sin(long)*radius
			o.Z = high / 256.0
			//o.Vx = (rand.Float64() - 0.5) * config.Velo * 2.0 * math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			//o.Vy = (rand.Float64() - 0.5) * config.Velo * 2.0 * math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			o.Vx = math.Cos(long+math.Pi/2.0) * config.Velo * 2.0 //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			o.Vy = math.Sin(long+math.Pi/2.0) * config.Velo * 2.0 //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			o.Vz = (rand.Float64() - 0.5) * config.Velo * 2.0 / 256.0
			o.Size = 1
			o.Mass = rand.Float64() * config.Mass
			o.Id = i // rand.Int()
			o.Stat = 1
			allMass += o.Mass
		}
	case 3: //球形 随机经度 随机半径 随机高度*sin(半径)
		for i := 0; i < num; i++ {
			o := &oList[i]
			long := rand.Float64() * math.Pi * 2
			radius := math.Sqrt(rand.Float64() * (config.Wide / 2.0 * config.Wide / 2.0))
			o.X, o.Y = math.Cos(long)*radius, math.Sin(long)*radius
			o.Z = (0.5 - rand.Float64()) * math.Sqrt(config.Wide*config.Wide/4.0-radius*radius) * 2.0
			o.Vx = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Vy = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Vz = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Size = 1
			o.Mass = rand.Float64() * config.Mass
			o.Id = i // rand.Int()
			o.Stat = 1
			allMass += o.Mass
		}
	default:
	}
	// 如果配置了恒星，将最后一个设置为恒星
	if config.Eternal != 0.0 {
		eternalId := num - 1
		eternalOrb := &oList[eternalId]
		allMass += config.Eternal - eternalOrb.Mass
		eternalOrb.Mass = config.Eternal
		eternalOrb.Id = eternalId //rand.Int()
		eternalOrb.X, eternalOrb.Y, eternalOrb.Z = 0, 0, 0
		eternalOrb.Vx, eternalOrb.Vy, eternalOrb.Vz = 0, 0, 0

	}
	return oList
}

// 所有天体运动一次
func UpdateOrbs(oList []Orb, nStep int) int {
	thelen := len(oList)
	nCount = 0
	//var vm massVary
	for i := 0; i < thelen; i++ {
		oList[i].idx = i
		go oList[i].Update(oList, c)
	}
	for {
		if nCount >= thelen {
			break
		}
		//log.Println("start waiting")
		select {

		case <-c:
			nCount += 1
		case ce := <-crashList:
			// 接受质量变化信息
			target := &oList[ce.idx]
			o := &oList[ce.crashedIdx]
			target.Mass += o.Mass
			o.Mass = 0
			target.Vx = (target.Mass*target.Vx + o.Mass*o.Vx) / target.Mass
			target.Vy = (target.Mass*target.Vy + o.Mass*o.Vy) / target.Mass
			target.Vz = (target.Mass*target.Vz + o.Mass*o.Vz) / target.Mass
			target.Size += 1
			//log.Println("the vary got:", ce)
			nCrashed++
			break
		}
		//log.Println("read once: val,nCount=", val, nCount)
	}
	return thelen * thelen
}

// 天体运动一次
func (o *Orb) Update(oList []Orb, c chan<- int) {
	// 先把位置移动起来，再计算环境中的加速度，再更新速度，为了更好地解决并行计算数据同步问题
	if o.Stat == 1 {
		aAll := o.CalcGravityAll(oList)
		o.X += o.Vx
		o.Y += o.Vy
		o.Z += o.Vz
		o.Vx += aAll.Ax
		o.Vy += aAll.Ay
		o.Vz += aAll.Az
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
	c <- o.Id //len(oList)
}

// 计算天体受到的总体引力
func (o *Orb) CalcGravityAll(oList []Orb) Acc {
	var gAll Acc
	for i := 0; i < len(oList); i++ {
		//c <- 1
		target := &oList[i]
		if target.Id == o.Id || target.Stat != 1 || o.Stat != 1 {
			continue
		}

		dist := o.CalcDist(target)

		// 距离太近，被撞
		isTooNearly := dist*dist < MIN_CRITICAL_DIST*MIN_CRITICAL_DIST
		// 速度太快，被撕裂 me ripped by ta
		isMeRipped := dist*dist < (o.Vx*o.Vx+o.Vy*o.Vy+o.Vz*o.Vz)*10

		if isTooNearly || isMeRipped {

			// 碰撞机制 非弹性碰撞 动量守恒 m1v1+m2v2=(m1+m2)v
			if o.Mass < target.Mass || isMeRipped {
				//log.Println(o.Id, "crashed by", target.Id, "isTooNearly", isTooNearly, isMeRipped, "me=", o, "ta=", target)
				//target.Mass += target.Mass
				//target.Vx = (target.Mass*target.Vx + o.Mass*o.Vx) / target.Mass
				//target.Vy = (target.Mass*target.Vy + o.Mass*o.Vy) / target.Mass
				//target.Vz = (target.Mass*target.Vz + o.Mass*o.Vz) / target.Mass
				//target.Size += 1
				// 碰撞对方的质量改变交给主goroutine，这里发送信息，不做修改操作
				crashList <- crashEvent{target.idx, o.idx}
				//o.Mass = 0
				o.Stat = 2
			} else {
				//log.Println(o.Id, "crashed", target.Id, "isTooNearly", isTooNearly, isMeRipped, "me=", o, "ta=", target)
				// 碰撞后速度 v = (m1v1+m2v2)/(m1+m2)
				//由于并发数据分离，当前goroutine只允许操作当前orb,不允许操作别的orb，所以不允许操作ta的数据
				//o.Mass += target.Mass
				//在轮训时可能有多个o crashed ta,但是只有一个o crashed by ta
				//o.Vx = (target.Mass*target.Vx + o.Mass*o.Vx) / o.Mass
				//o.Vy = (target.Mass*target.Vy + o.Mass*o.Vy) / o.Mass
				//o.Vz = (target.Mass*target.Vz + o.Mass*o.Vz) / o.Mass
				//o.Size += 1
				//target.Mass = 0
				//target.Stat = 2
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

// 清理orbList中的垃圾
func ClearOrbList(oList []Orb) []Orb {
	allWC = 0
	var alive int = len(oList)
	for i := 0; i < len(oList); i++ {
		if oList[i].Stat != 1 {
			oList = append(oList[:i], oList[i+1:]...)
			i--
			alive--
		} else {
			allWC += oList[i].Mass
		}
	}
	//log.Println("when clear alive=", alive)
	clearTimes++
	return oList
}

func ShowMonitorInfo() {
	log.Printf("maxVelo=%.6g %.6g %.6g maxAcc=%.6g %.6g %.6g maxMass=%d %e allMass=%e\n", maxVeloX, maxVeloY, maxVeloZ, maxAccX, maxAccY, maxAccZ, maxMassId, maxMass, allWC)
}
func GetClearTimes() int {
	return clearTimes
}
func GetCalcTimes() int {
	return nCount
}
func GetCrashed() int {
	return nCrashed
}
